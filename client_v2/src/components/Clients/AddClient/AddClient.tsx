import React, { useCallback, useEffect, useMemo, ChangeEvent } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import { IOption } from 'panel/lib/helpers/utils';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { PageLoader } from 'panel/common/ui/Loader';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';
import { RootState, Client } from 'panel/initialState';
import {
    updateClientFormField,
    clearClientForm,
    saveClient,
    initClientForm,
    buildFormPayload,
    setFormErrors,
} from 'panel/actions/clientForm';
import { getClients } from 'panel/actions';
import { validateUpstreams } from 'panel/helpers/validators';
import { RoutePath, Paths } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import { ClientsHeader } from './blocks/ClientsHeader';
import { Identifiers } from './blocks/Identifiers/Identifiers';

import s from './AddClient.module.pcss';

export const AddClient = () => {
    const dispatch = useDispatch();
    const navigate = useNavigate();
    const location = useLocation();
    const { clientName: urlClientName } = useParams<{ clientName?: string }>();
    const form = useSelector((state: RootState) => state.clientForm);
    const dashboard = useSelector((state: RootState) => state.dashboard);
    const clients = dashboard?.clients || [];
    const isEdit = form.mode === 'edit';

    useEffect(() => {
        dispatch(getClients());
    }, []);

    useEffect(() => {
        const searchParams = new URLSearchParams(location.search);
        const id = searchParams.get('id');

        if (!id || form.mode !== 'add') {
            return;
        }

        dispatch(updateClientFormField({ field: 'ids', value: [id] }));
    }, [location.search]);

    // When on the edit route and clients are loaded, initialize the form
    // from the URL param. This handles page refreshes on the edit page.
    useEffect(() => {
        if (!urlClientName || !clients.length) {
            return;
        }

        if (form.mode !== 'add' || form.originalName) {
            return;
        }

        const decodedName = decodeURIComponent(urlClientName);
        const client = clients.find((c: Client) => c.name === decodedName);

        if (!client) {
            dispatch(clearClientForm());
            navigate(Paths.Clients, { replace: true });
            return;
        }

        dispatch(initClientForm(buildFormPayload(client)));
    }, [urlClientName, clients, form.mode, form.originalName, dispatch, navigate]);

    const handleCancel = useCallback(() => {
        dispatch(clearClientForm());
        navigate(Paths.Clients);
    }, [dispatch, navigate]);

    const handleSave = useCallback(async () => {
        const err = validateUpstreams(form.upstreams);
        if (err) {
            dispatch(setFormErrors({ upstreams: err }));
            return;
        }
        const result = await dispatch(saveClient());
        if (result) {
            navigate(Paths.Clients);
        }
    }, [dispatch, navigate, form.upstreams]);

    const handleUseGlobalSettings = useCallback((e: ChangeEvent<HTMLInputElement>) => {
        dispatch(
            updateClientFormField({
                field: 'use_global_settings',
                value: e.target.checked,
            }),
        );
    }, []);

    const handleUpstreamsCacheEnabled = useCallback((e: ChangeEvent<HTMLInputElement>) => {
        dispatch(
            updateClientFormField({
                field: 'upstreams_cache_enabled',
                value: e.target.checked,
            }),
        );
    }, []);

    const handleUpstreamsBlur = useCallback(() => {
        const err = validateUpstreams(form.upstreams);
        if (err) {
            dispatch(setFormErrors({ upstreams: err }));
        }
    }, [dispatch, form.upstreams]);

    const handleUpstreamsChange = useCallback(
        (e: ChangeEvent<HTMLTextAreaElement>) => {
            dispatch(updateClientFormField({ field: 'upstreams', value: e.target.value }));
        },
        [dispatch],
    );

    const tagOptions = useMemo(
        () =>
            (dashboard.supportedTags || []).map((tag: string) => ({
                label: tag,
                value: tag,
            })),
        [dashboard.supportedTags],
    );
    const tagValue = useMemo(
        () => form.tags.map((t: string) => ({ label: t, value: t })),
        [form.tags],
    );

    const handleTagChange = useCallback((selected: IOption<string> | IOption<string>[] | null) => {
        const tags = Array.isArray(selected) ? selected.map((s) => s.value) : [];
        dispatch(updateClientFormField({ field: 'tags', value: tags }));
    }, []);

    const protectionRoute = isEdit ? RoutePath.ClientsEditProtection : RoutePath.ClientsProtection;
    const blockedServicesRoute = isEdit
        ? RoutePath.ClientsEditBlockedServices
        : RoutePath.ClientsBlockedServices;
    const routeProps = isEdit ? { clientName: encodeURIComponent(form.originalName) } : undefined;

    return (
        <div className={cn(theme.layout.container, s.containerOverride)} data-testid="client-form">
            <div
                className={cn(
                    theme.layout.containerIn,
                    theme.layout.containerIn_one_col,
                    s.pageWrapper,
                )}
            >
                <ClientsHeader currentTitle={intl.getMessage('clients_add')} />

                {form.processingSave && <PageLoader />}

                {!form.processingSave && (
                    <>
                        <div className={s.fieldGroupInput}>
                            <Input
                                id="client-name"
                                data-testid="client-form-name"
                                type="text"
                                value={form.name}
                                onChange={(e) =>
                                    dispatch(
                                        updateClientFormField({
                                            field: 'name',
                                            value: e.target.value,
                                        }),
                                    )
                                }
                                placeholder={intl.getMessage('clients_add_default_name')}
                                label={intl.getMessage('clients_add_name')}
                                size="large"
                                error={!!form.formErrors.name}
                                errorMessage={
                                    typeof form.formErrors.name === 'string'
                                        ? form.formErrors.name
                                        : undefined
                                }
                            />
                        </div>

                        <div className={s.section}>
                            <Identifiers />
                        </div>

                        <div className={s.section}>
                            <div className={cn(theme.text.t2, theme.text.semibold, s.fieldLabel)}>
                                {intl.getMessage('clients_tags')}
                            </div>
                            <div className={cn(theme.text.t3, s.fieldDesc)}>
                                {intl.getMessage('clients_tags_desc', {
                                    a: (text: string) => (
                                        <a
                                            href="https://adguard-dns.io/kb/general/dns-filtering-syntax/#ctag"
                                            target="_blank"
                                            rel="noopener noreferrer"
                                        >
                                            {text}
                                        </a>
                                    ),
                                })}
                            </div>
                            <Select
                                options={tagOptions}
                                value={tagValue}
                                onChange={handleTagChange}
                                isMulti
                                size="responsive"
                                height="big"
                                closeMenuOnSelect={false}
                            />
                        </div>

                        <h2
                            className={cn(
                                theme.layout.subtitle,
                                theme.title.h5,
                                theme.title.h4_tablet,
                            )}
                        >
                            {intl.getMessage('settings')}
                        </h2>

                        <SwitchGroup
                            id="use-global-settings"
                            title={intl.getMessage('clients_use_global_settings')}
                            description={intl.getMessage('clients_use_global_settings_desc')}
                            checked={form.use_global_settings}
                            onChange={handleUseGlobalSettings}
                        />

                        <Link
                            to={protectionRoute}
                            props={routeProps}
                            className={cn(s.navLink, {
                                [s.navLinkDisabled]: form.use_global_settings,
                            })}
                            disabled={form.use_global_settings}
                        >
                            <div className={s.navItem}>
                                <div className={s.navItemContent}>
                                    <div
                                        className={cn(
                                            theme.text.t2,
                                            theme.text.semibold,
                                            s.navTitle,
                                        )}
                                    >
                                        {intl.getMessage('clients_protection')}
                                    </div>
                                    <div className={cn(theme.text.t3, s.navDesc)}>
                                        {intl.getMessage('clients_protection_desc')}
                                    </div>
                                </div>
                                <Icon icon="arrow" color="gray" />
                            </div>
                        </Link>

                        <Link
                            to={blockedServicesRoute}
                            props={routeProps}
                            className={cn(s.navLink, {
                                [s.navLinkDisabled]: form.use_global_settings,
                            })}
                            disabled={form.use_global_settings}
                        >
                            <div className={s.navItem}>
                                <div className={s.navItemContent}>
                                    <div
                                        className={cn(
                                            theme.text.t2,
                                            theme.text.semibold,
                                            s.navTitle,
                                        )}
                                    >
                                        {intl.getMessage('blocked_services')}
                                    </div>
                                    <div className={cn(theme.text.t3, s.navDesc)}>
                                        {intl.getMessage('blocked_services_desc')}
                                    </div>
                                </div>
                                <Icon icon="arrow" color="gray" />
                            </div>
                        </Link>

                        <h2
                            className={cn(
                                theme.layout.subtitle,
                                theme.title.h5,
                                theme.title.h4_tablet,
                                {
                                    [s.disabledText]: form.use_global_settings,
                                },
                            )}
                        >
                            {intl.getMessage('upstream_dns_servers_title')}
                        </h2>

                        <div className={s.section}>
                            <Textarea
                                id="client-upstreams"
                                value={form.upstreams}
                                onChange={handleUpstreamsChange}
                                onBlur={handleUpstreamsBlur}
                                placeholder={intl.getMessage('upstream_dns_placeholder')}
                                label={intl.getMessage('clients_upstreams_desc')}
                                rows={4}
                                disabled={form.use_global_settings}
                                errorMessage={
                                    typeof form.formErrors.upstreams === 'string'
                                        ? form.formErrors.upstreams
                                        : undefined
                                }
                            />
                        </div>

                        <SwitchGroup
                            id="use-dns-cache"
                            title={intl.getMessage('clients_use_dns_cache')}
                            checked={form.upstreams_cache_enabled}
                            onChange={handleUpstreamsCacheEnabled}
                            disabled={form.use_global_settings}
                        >
                            {form.upstreams_cache_enabled && (
                                <Input
                                    id="dns-cache-size"
                                    type="text"
                                    value={String(form.upstreams_cache_size)}
                                    onChange={(e) =>
                                        dispatch(
                                            updateClientFormField({
                                                field: 'upstreams_cache_size',
                                                value: Number(e.target.value) || 0,
                                            }),
                                        )
                                    }
                                    placeholder={intl.getMessage(
                                        'clients_dns_cache_size_placeholder',
                                    )}
                                    label={intl.getMessage('clients_dns_cache_size')}
                                    size="large"
                                />
                            )}
                        </SwitchGroup>

                        <div className={s.actions}>
                            <Button
                                variant="primary"
                                size="small"
                                onClick={handleSave}
                                disabled={form.processingSave}
                                data-testid="client-form-save"
                                className={s.actionButton}
                            >
                                {intl.getMessage('save_btn')}
                            </Button>
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={handleCancel}
                                data-testid="client-form-cancel"
                                className={s.actionButton}
                            >
                                {intl.getMessage('cancel_btn')}
                            </Button>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};
