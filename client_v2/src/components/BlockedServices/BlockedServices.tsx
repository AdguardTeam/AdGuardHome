import { createSignal, createEffect, createMemo, onMount, Show, For } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { updateClientFormField, clientFormState } from 'panel/stores/clientForm';
import theme from 'panel/lib/theme';
import {
    getBlockedServices,
    getAllBlockedServices,
    updateBlockedServices,
    servicesState,
} from 'panel/stores/services';

import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { GroupFilter } from './GroupFilter';
import { ServiceRow } from './ServiceRow';
import { NothingFound } from './NothingFound';

import s from './BlockedServices.module.pcss';
import { RoutePath, type RoutePathKey } from '../Routes/Paths';

type WebService = {
    id: string;
    name: string;
    icon_svg: string;
    group_id: string;
    rules: string[];
};

type Props = {
    clientScope?: boolean;
    class?: string;
    breadcrumbs?: {
        parentLinks: {
            path: RoutePathKey;
            title: string;
            props?: Partial<Record<string, string | number>>;
        }[];
        currentTitle: string;
    };
};

export const BlockedServices = (props: Props) => {
    const [search, setSearch] = createSignal('');
    const [groupFilter, setGroupFilter] = createSignal<string[]>([]);
    const [togglingId, setTogglingId] = createSignal<string | null>(null);

    onMount(() => {
        if (!props.clientScope) {
            getBlockedServices();
            getAllBlockedServices();
        } else {
            getAllBlockedServices();
        }
    });

    createEffect(() => {
        if (!servicesState.processingSet) {
            setTogglingId(null);
        }
    });

    const blockedSet = createMemo(() => {
        if (props.clientScope) {
            return new Set<string>(clientFormState.blocked_services || []);
        }
        return new Set<string>(servicesState.list?.ids || []);
    });

    const serviceGroupMap = createMemo(() => {
        const map = new Map<string, string>();
        const allServices = servicesState.allServices;
        if (!allServices || allServices.length === 0) {
            return map;
        }
        (allServices as WebService[]).forEach((service) => {
            if (service.group_id) {
                map.set(service.id, service.group_id);
            }
        });
        return map;
    });

    const filteredServices = createMemo(() => {
        const allServices = servicesState.allServices;
        if (!allServices || allServices.length === 0) {
            return [];
        }
        let filtered = [...(allServices as WebService[])];
        const gf = groupFilter();
        if (gf.length > 0) {
            const selected = new Set(gf);
            const sgMap = serviceGroupMap();
            filtered = filtered.filter((service) => {
                const groupId = sgMap.get(service.id);
                return groupId && selected.has(groupId);
            });
        }
        const term = search().trim().toLowerCase();
        if (term) {
            filtered = filtered.filter(
                (service) =>
                    service.name.toLowerCase().includes(term) ||
                    service.id.toLowerCase().includes(term),
            );
        }
        filtered.sort((a, b) => a.name.localeCompare(b.name));
        return filtered;
    });

    const handleToggleGroup = (groupId: string) => {
        setGroupFilter((current) =>
            current.includes(groupId)
                ? current.filter((g) => g !== groupId)
                : [...current, groupId],
        );
    };

    const handleToggleService = (serviceId: string, checked: boolean) => {
        if (props.clientScope) {
            const currentIds: string[] = clientFormState.blocked_services || [];
            const newIds = checked
                ? [...currentIds, serviceId]
                : currentIds.filter((id: string) => id !== serviceId);
            updateClientFormField('blocked_services', newIds);
            return;
        }
        setTogglingId(serviceId);
        const currentIds = servicesState.list?.ids || [];
        const newIds = checked
            ? [...currentIds, serviceId]
            : currentIds.filter((id: string) => id !== serviceId);
        updateBlockedServices({ ids: newIds, schedule: servicesState.list?.schedule });
    };

    const handleSearchChange = (e: Event) => {
        setSearch((e.target as HTMLInputElement).value);
    };

    const handleSearchClear = () => {
        setSearch('');
    };

    const isInitialLoading = () =>
        (servicesState.allServices == null || servicesState.allServices.length === 0) &&
        (servicesState.processingAll || servicesState.processing);
    const isGloballyDisabled = () =>
        props.clientScope ? clientFormState.use_global_blocked_services : false;

    const getScheduleRoute = () => {
        if (!props.clientScope) {
            return RoutePath.InactivitySchedule;
        }
        return clientFormState.mode === 'edit'
            ? RoutePath.ClientsEditSchedule
            : RoutePath.ClientsSchedule;
    };

    const scheduleRouteProps = () =>
        props.clientScope && clientFormState.mode === 'edit'
            ? { clientName: encodeURIComponent(clientFormState.originalName) }
            : undefined;

    return (
        <Show when={!isInitialLoading()}>
            <div class={cn(theme.layout.container, props.class)}>
                <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                    <Show when={!props.clientScope && !props.breadcrumbs}>
                        <div class={s.header}>
                            <h1
                                class={cn(
                                    theme.layout.title,
                                    theme.title.h4,
                                    theme.title.h3_tablet,
                                )}
                                data-testid="blocked-services-title"
                            >
                                {intl.getMessage('blocked_services')}
                            </h1>
                            <p class={s.description}>{intl.getMessage('blocked_services_desc')}</p>
                        </div>
                    </Show>

                    <Show when={props.breadcrumbs}>
                        <div class={s.breadcrumbs}>
                            <Breadcrumbs
                                parentLinks={props.breadcrumbs!.parentLinks}
                                currentTitle={props.breadcrumbs!.currentTitle}
                            />
                        </div>
                        <h1
                            class={cn(
                                theme.layout.title,
                                theme.title.h4,
                                theme.title.h3_tablet,
                                s.clientsTitle,
                            )}
                        >
                            {intl.getMessage('blocked_services')}
                        </h1>
                        <p class={s.description}>{intl.getMessage('blocked_services_desc')}</p>
                    </Show>

                    <Link to={getScheduleRoute()} props={scheduleRouteProps()} class={s.navItem} data-testid="blocked-services-schedule-link">
                        <div class={s.navItemContent}>
                            <div class={s.navItemTitle}>
                                {intl.getMessage('inactivity_schedule')}
                            </div>
                            <div class={s.navItemDesc}>
                                {intl.getMessage('inactivity_schedule_desc')}
                            </div>
                        </div>
                        <Icon icon="arrow" color="gray" />
                    </Link>

                    <div class={s.search}>
                        <Input
                            id="blocked-services-search"
                            data-testid="blocked-services-search"
                            type="text"
                            value={search()}
                            onInput={handleSearchChange}
                            placeholder={intl.getMessage('search_placeholder')}
                            prefixIcon={<Icon icon="search" class={s.searchIcon} color="gray" />}
                            suffixIcon={
                                <Show when={search()}>
                                    <button
                                        type="button"
                                        onClick={handleSearchClear}
                                        class={s.clearButton}
                                        aria-label={intl.getMessage('clear_btn')}
                                        data-testid="blocked-services-search-clear"
                                    >
                                        <Icon icon="cross" color="gray" />
                                    </button>
                                </Show>
                            }
                        />
                    </div>

                    <GroupFilter
                        groups={servicesState.allGroups || []}
                        activeGroups={groupFilter()}
                        onToggleGroup={handleToggleGroup}
                        data-testid="blocked-services-groups"
                    />

                    <div class={s.servicesList} data-testid="blocked-services-list">
                        <Show
                            when={filteredServices().length === 0}
                            fallback={
                                <For each={filteredServices()}>
                                    {(service) => (
                                        <ServiceRow
                                            id={service.id}
                                            name={service.name}
                                            iconSvg={service.icon_svg}
                                            checked={blockedSet().has(service.id)}
                                            disabled={
                                                isGloballyDisabled() || togglingId() === service.id
                                            }
                                            onChange={handleToggleService}
                                        />
                                    )}
                                </For>
                            }
                        >
                            <NothingFound />
                        </Show>
                    </div>
                </div>
            </div>
        </Show>
    );
};
