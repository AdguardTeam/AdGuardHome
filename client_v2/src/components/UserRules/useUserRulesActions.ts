import { useMemo, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { useDispatch, useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import {
    getClients,
    initSettings,
    toggleBlocking,
    toggleBlockingForClient,
    toggleSetting,
} from 'panel/actions';
import { updateClient } from 'panel/actions/clients';
import {
    checkHost,
    getFilteringStatus,
    setRules,
    toggleFilterStatus,
} from 'panel/actions/filtering';
import { updateRewrite, deleteRewrite, addRewrite, getRewritesList } from 'panel/actions/rewrites';
import { getBlockedServices, updateBlockedServices } from 'panel/actions/services';
import { addSuccessToast, createUndoToast } from 'panel/actions/toasts';
import { BLOCK_ACTIONS, MODAL_TYPE, SPECIAL_FILTER_ID } from 'panel/helpers/constants';
import { delay, splitByNewLine } from 'panel/helpers/helpers';
import { Client, RootState } from 'panel/initialState';
import { openModal } from 'panel/reducers/modals';
import type { AppDispatch } from 'panel/store/types';

import {
    CLIENT_SCOPED_ACTIONS,
    findMatchedBlockedService,
    findMatchedRewrite,
    findPersistentClient,
    getEffectiveBlockedServices,
    getEffectiveClientProtectionSettings,
    getPrimaryRule,
} from './helpers';
import { CheckFormValues, CheckResultData, ResultActionKind, RewriteEntry } from './types';

const EMPTY_REWRITE: RewriteEntry = {
    domain: '',
    answer: '',
    enabled: false,
};

const RECHECK_DELAY_MS = 500;

type UseUserRulesActionsParams = {
    checkResult: CheckResultData | null;
    filters: RootState['filtering']['filters'];
    whitelistFilters: RootState['filtering']['whitelistFilters'];
    filteringEnabled: boolean;
    settingsList?: RootState['settings']['settingsList'];
    persistentClients: Client[];
    rewritesList: RootState['rewrites']['list'];
    services: RootState['services'];
    lastSubmittedCheck: CheckFormValues | null;
    setIsResultVisible: Dispatch<SetStateAction<boolean>>;
};

type UseUserRulesActionsResult = {
    currentRewrite: RewriteEntry;
    setCurrentRewrite: Dispatch<SetStateAction<RewriteEntry>>;
    matchedRewrite: RewriteEntry | null;
    hiddenActionKinds: ResultActionKind[];
    handleAction: (action: ResultActionKind) => Promise<void>;
    handleRewriteUpdate: (update: RewriteEntry) => Promise<boolean>;
    handleRewriteDelete: () => Promise<boolean>;
    openEditRewriteModal: () => void;
    openDeleteRewriteModal: () => void;
    resetCurrentRewrite: () => void;
};

export const useUserRulesActions = ({
    checkResult,
    filters,
    whitelistFilters,
    filteringEnabled,
    settingsList,
    persistentClients,
    rewritesList,
    services,
    lastSubmittedCheck,
    setIsResultVisible,
}: UseUserRulesActionsParams): UseUserRulesActionsResult => {
    const dispatch = useDispatch<AppDispatch>();
    const userRules = useSelector((state: RootState) => state.filtering.userRules);
    const [currentRewrite, setCurrentRewrite] = useState<RewriteEntry>(EMPTY_REWRITE);

    const matchedRewrite = useMemo(
        () => findMatchedRewrite(rewritesList, checkResult),
        [rewritesList, checkResult],
    );

    const resolvedClient = useMemo(
        () => findPersistentClient(persistentClients, lastSubmittedCheck?.client),
        [persistentClients, lastSubmittedCheck?.client],
    );

    const hiddenActionKinds = useMemo(() => {
        if (!lastSubmittedCheck?.client || resolvedClient) {
            return [];
        }

        return CLIENT_SCOPED_ACTIONS;
    }, [resolvedClient, lastSubmittedCheck?.client]);

    const runWithClosedResult = async <T>(callback: () => Promise<T>): Promise<T> => {
        setIsResultVisible(false);

        return callback();
    };

    const recheckCurrentTarget = async () => {
        if (!lastSubmittedCheck) {
            return;
        }

        await delay(RECHECK_DELAY_MS);

        await dispatch(
            checkHost({
                name: lastSubmittedCheck.hostname,
                client: lastSubmittedCheck.client || undefined,
                qtype: lastSubmittedCheck.qtype || undefined,
            }),
        );

        setIsResultVisible(true);
    };

    const updateResolvedClient = async (transform: (client: Client) => Client | null) => {
        if (!resolvedClient) {
            return false;
        }

        const updatedClient = transform(resolvedClient);

        if (!updatedClient) {
            return false;
        }

        return dispatch(
            updateClient(updatedClient, resolvedClient.name, {
                showToast: false,
                toggleModal: false,
            }),
        );
    };

    const performWithUndo = async (params: {
        perform: () => Promise<boolean | undefined | void>;
        message: string;
        undo: () => Promise<boolean | undefined | void>;
        refresh: () => Promise<void>;
    }) => {
        return runWithClosedResult(async () => {
            const ok = await params.perform();

            if (ok === false) {
                return false;
            }

            dispatch(
                addSuccessToast(
                    createUndoToast(params.message, intl.getMessage('notify_undo'), async () => {
                        const didUndo = await params.undo();

                        if (didUndo) {
                            await params.refresh();
                        }
                    }),
                ),
            );
            await recheckCurrentTarget();

            return true;
        });
    };

    const refreshSettingsAndClients = async () => {
        await dispatch(initSettings());
        await dispatch(getClients());
    };

    const handleRuleToggle = async (type: (typeof BLOCK_ACTIONS)[keyof typeof BLOCK_ACTIONS]) => {
        if (!checkResult?.hostname || !lastSubmittedCheck) {
            return;
        }

        const primaryRule = getPrimaryRule(checkResult);
        const matchedCustomRule =
            !lastSubmittedCheck.client &&
            primaryRule?.filter_list_id === SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES
                ? primaryRule.text
                : undefined;

        let action;

        if (lastSubmittedCheck.client) {
            action = toggleBlockingForClient(type, checkResult.hostname, lastSubmittedCheck.client);
        } else if (matchedCustomRule) {
            action = toggleBlocking(
                type,
                checkResult.hostname,
                undefined,
                undefined,
                matchedCustomRule,
            );
        } else {
            action = toggleBlocking(type, checkResult.hostname);
        }

        await runWithClosedResult(async () => {
            const ok = await dispatch(action);

            if (ok === false) {
                return false;
            }

            await recheckCurrentTarget();

            return true;
        });
    };

    const handleDisableSafeBrowsing = async () => {
        const clientSnapshot = resolvedClient;

        await performWithUndo({
            perform: () =>
                lastSubmittedCheck?.client
                    ? updateResolvedClient((client) => {
                          const effectiveSettings = getEffectiveClientProtectionSettings({
                              client,
                              globalFilteringEnabled: filteringEnabled,
                              settingsList,
                          });

                          if (!effectiveSettings) {
                              return null;
                          }

                          return {
                              ...client,
                              ...effectiveSettings,
                              use_global_settings: false,
                              safebrowsing_enabled: false,
                          };
                      })
                    : dispatch(toggleSetting('safebrowsing', true)),
            message: intl.getMessage('user_rules_browsing_security_disabled'),
            undo: () =>
                lastSubmittedCheck?.client && clientSnapshot
                    ? dispatch(
                          updateClient(
                              { ...clientSnapshot, safebrowsing_enabled: true },
                              clientSnapshot.name,
                              { showToast: false, toggleModal: false },
                          ),
                      )
                    : dispatch(toggleSetting('safebrowsing', false)),
            refresh: refreshSettingsAndClients,
        });
    };

    const handleDisableParental = async () => {
        const clientSnapshot = resolvedClient;

        await performWithUndo({
            perform: () =>
                lastSubmittedCheck?.client
                    ? updateResolvedClient((client) => {
                          const effectiveSettings = getEffectiveClientProtectionSettings({
                              client,
                              globalFilteringEnabled: filteringEnabled,
                              settingsList,
                          });

                          if (!effectiveSettings) {
                              return null;
                          }

                          return {
                              ...client,
                              ...effectiveSettings,
                              use_global_settings: false,
                              parental_enabled: false,
                          };
                      })
                    : dispatch(toggleSetting('parental', true)),
            message: intl.getMessage('user_rules_parental_control_disabled'),
            undo: () =>
                lastSubmittedCheck?.client && clientSnapshot
                    ? dispatch(
                          updateClient(
                              { ...clientSnapshot, parental_enabled: true },
                              clientSnapshot.name,
                              { showToast: false, toggleModal: false },
                          ),
                      )
                    : dispatch(toggleSetting('parental', false)),
            refresh: refreshSettingsAndClients,
        });
    };

    const handleDisableSafeSearch = async () => {
        const currentSafeSearch = settingsList?.safesearch;

        if (!lastSubmittedCheck?.client && !currentSafeSearch) {
            return;
        }

        const clientSnapshot = resolvedClient;

        await performWithUndo({
            perform: () =>
                lastSubmittedCheck?.client
                    ? updateResolvedClient((client) => {
                          const effectiveSettings = getEffectiveClientProtectionSettings({
                              client,
                              globalFilteringEnabled: filteringEnabled,
                              settingsList,
                          });

                          if (!effectiveSettings) {
                              return null;
                          }

                          const safeSearch = {
                              ...effectiveSettings.safe_search,
                              enabled: false,
                          };

                          return {
                              ...client,
                              ...effectiveSettings,
                              use_global_settings: false,
                              safe_search: safeSearch,
                              safesearch_enabled: safeSearch.enabled,
                          };
                      })
                    : dispatch(
                          toggleSetting('safesearch', { ...currentSafeSearch, enabled: false }),
                      ),
            message: intl.getMessage('user_rules_safe_search_disabled'),
            undo: () =>
                lastSubmittedCheck?.client && clientSnapshot
                    ? dispatch(
                          updateClient(
                              {
                                  ...clientSnapshot,
                                  safe_search: { ...clientSnapshot.safe_search, enabled: true },
                                  safesearch_enabled: true,
                              },
                              clientSnapshot.name,
                              { showToast: false, toggleModal: false },
                          ),
                      )
                    : dispatch(
                          toggleSetting('safesearch', { ...currentSafeSearch, enabled: true }),
                      ),
            refresh: refreshSettingsAndClients,
        });
    };

    const handleDisableFilter = async () => {
        const filterId = getPrimaryRule(checkResult)?.filter_list_id;
        if (filterId === undefined) {
            return;
        }

        const matchedFilter =
            filters.find((filter) => filter.id === filterId) ??
            whitelistFilters.find((filter) => filter.id === filterId);
        if (!matchedFilter) {
            return;
        }

        const isWhitelist = whitelistFilters.some((filter) => filter.id === filterId);

        await performWithUndo({
            perform: () =>
                dispatch(
                    toggleFilterStatus(
                        matchedFilter.url,
                        { name: matchedFilter.name, url: matchedFilter.url, enabled: false },
                        isWhitelist,
                    ),
                ),
            message: intl.getMessage('user_rules_filter_was_disabled', {
                value: matchedFilter.name,
            }),
            undo: () =>
                dispatch(
                    toggleFilterStatus(
                        matchedFilter.url,
                        { name: matchedFilter.name, url: matchedFilter.url, enabled: true },
                        isWhitelist,
                    ),
                ),
            refresh: () => recheckCurrentTarget(),
        });
    };

    const handleAllowBlockedService = async () => {
        const matchedService = findMatchedBlockedService(services.allServices, checkResult);

        if (!matchedService) {
            return;
        }

        const previousBlockedServiceIds = [...(services.list.ids || [])];
        const clientSnapshot = resolvedClient;

        await performWithUndo({
            perform: () =>
                resolvedClient
                    ? updateResolvedClient((client) => {
                          const effectiveBlockedServices = getEffectiveBlockedServices(
                              client,
                              services.list,
                          );

                          if (!effectiveBlockedServices) {
                              return null;
                          }

                          return {
                              ...client,
                              ...effectiveBlockedServices,
                              use_global_blocked_services: false,
                              blocked_services: effectiveBlockedServices.blocked_services.filter(
                                  (id) => id !== matchedService.id,
                              ),
                          };
                      })
                    : dispatch(
                          updateBlockedServices({
                              ...services.list,
                              ids: (services.list.ids || []).filter(
                                  (id: string) => id !== matchedService.id,
                              ),
                          }),
                      ),
            message: intl.getMessage('user_rules_service_allowed', { value: matchedService.name }),
            undo: () =>
                clientSnapshot
                    ? dispatch(
                          updateClient(
                              {
                                  ...clientSnapshot,
                                  blocked_services: [...previousBlockedServiceIds],
                              },
                              clientSnapshot.name,
                              { showToast: false, toggleModal: false },
                          ),
                      )
                    : dispatch(
                          updateBlockedServices({
                              ...services.list,
                              ids: previousBlockedServiceIds,
                          }),
                      ),
            refresh: async () => {
                await dispatch(getBlockedServices());
                await dispatch(getClients());
            },
        });
    };

    const handleRewriteUpdate = async (update: RewriteEntry) => {
        if (!currentRewrite.domain) {
            return false;
        }

        return runWithClosedResult(async () => {
            const ok = await dispatch(
                updateRewrite(
                    { target: currentRewrite, update },
                    { showToast: false, closeModal: false },
                ),
            );

            if (!ok) {
                return false;
            }

            dispatch(addSuccessToast(intl.getMessage('settings_notify_changes_saved')));
            await recheckCurrentTarget();

            return true;
        });
    };

    const handleRewriteDelete = async (rewrite?: RewriteEntry) => {
        const targetRewrite = rewrite || currentRewrite;

        if (!targetRewrite.domain) {
            return false;
        }

        return performWithUndo({
            perform: () =>
                dispatch(deleteRewrite(targetRewrite, { showToast: false })) as Promise<boolean>,
            message: intl.getMessage('user_rules_dns_rewrite_removed'),
            undo: async () => {
                await dispatch(
                    addRewrite({
                        domain: targetRewrite.domain,
                        answer: targetRewrite.answer,
                        enabled: targetRewrite.enabled,
                    }),
                );

                return true;
            },
            refresh: async () => {
                await dispatch(getRewritesList());
            },
        });
    };

    const resetCurrentRewrite = () => {
        setCurrentRewrite(EMPTY_REWRITE);
    };

    const openEditRewriteModal = () => {
        if (!matchedRewrite) {
            return;
        }

        setCurrentRewrite(matchedRewrite);
        dispatch(openModal(MODAL_TYPE.EDIT_REWRITE));
    };

    const openDeleteRewriteModal = () => {
        if (!matchedRewrite) {
            return;
        }

        handleRewriteDelete(matchedRewrite);
    };

    const handleRemoveRewriteRule = async () => {
        const primaryRule = getPrimaryRule(checkResult);

        if (!primaryRule?.text) {
            return;
        }

        const ruleToRemove = primaryRule.text;
        const originalRules = userRules || '';

        await performWithUndo({
            perform: () => {
                const currentRules = splitByNewLine(userRules || '');
                const updatedRules = currentRules.filter((rule: string) => rule !== ruleToRemove);

                return dispatch(setRules(`${updatedRules.join('\n')}\n`)) as Promise<boolean>;
            },
            message: intl.getMessage('user_rules_rule_removed'),
            undo: () => dispatch(setRules(originalRules)) as Promise<boolean>,
            refresh: async () => {
                await dispatch(getFilteringStatus());
            },
        });
    };

    const handleAction = async (action: ResultActionKind) => {
        switch (action) {
            case 'allow':
                await handleRuleToggle(BLOCK_ACTIONS.UNBLOCK);
                break;
            case 'block':
                await handleRuleToggle(BLOCK_ACTIONS.BLOCK);
                break;
            case 'disable-parental':
                await handleDisableParental();
                break;
            case 'disable-safebrowsing':
                await handleDisableSafeBrowsing();
                break;
            case 'disable-safesearch':
                await handleDisableSafeSearch();
                break;
            case 'disable-blocked-service':
                await handleAllowBlockedService();
                break;
            case 'disable-filter':
                await handleDisableFilter();
                break;
            case 'remove-rewrite-rule':
                await handleRemoveRewriteRule();
                break;
            case 'edit-rewrite':
            case 'delete-rewrite':
            default:
                break;
        }
    };

    return {
        currentRewrite,
        setCurrentRewrite,
        matchedRewrite,
        hiddenActionKinds,
        handleAction,
        handleRewriteUpdate,
        handleRewriteDelete,
        openEditRewriteModal,
        openDeleteRewriteModal,
        resetCurrentRewrite,
    };
};
