import React, { useEffect, useMemo, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';
import {
    getBlockedServices,
    getAllBlockedServices,
    updateBlockedServices,
} from 'panel/actions/services';

import { GroupFilter } from './GroupFilter';
import { ServiceRow } from './ServiceRow';
import { NothingFound } from './NothingFound';

import s from './BlockedServices.module.pcss';
import { RoutePath } from '../Routes/Paths';

type WebService = {
    id: string;
    name: string;
    icon_svg: string;
    group_id: string;
    rules: string[];
}

export const BlockedServices = () => {
    const dispatch = useDispatch();
    const services = useSelector((state: RootState) => state.services);
    const { processing, processingAll, processingSet, list, allServices, allGroups } = services;

    const [search, setSearch] = useState('');
    const [groupFilter, setGroupFilter] = useState<string[]>([]);

    useEffect(() => {
        dispatch(getBlockedServices());
        dispatch(getAllBlockedServices());
    }, [dispatch]);

    const blockedSet = useMemo(() => {
        return new Set<string>(list?.ids || []);
    }, [list]);

    const serviceGroupMap = useMemo(() => {
        const map = new Map<string, string>();

        if (!allServices) {
            return map;
        }

        allServices.forEach((service: WebService) => {
            if (service.group_id) {
                map.set(service.id, service.group_id);
            }
        });

        return map;
    }, [allServices]);

    const filteredServices = useMemo(() => {
        if (!allServices) {
            return [];
        }
        let filtered = allServices as WebService[];

        if (groupFilter.length > 0) {
            const selected = new Set(groupFilter);
            filtered = filtered.filter((service) => {
                const groupId = serviceGroupMap.get(service.id);
                return groupId && selected.has(groupId);
            });
        }

        const term = search.trim().toLowerCase();
        if (term) {
            filtered = filtered.filter(
                (service) =>
                    service.name.toLowerCase().includes(term) ||
                    service.id.toLowerCase().includes(term),
            );
        }

        return filtered;
    }, [allServices, search, groupFilter, serviceGroupMap]);

    const handleToggleGroup = (groupId: string) => {
        setGroupFilter((current) =>
            current.includes(groupId)
                ? current.filter((g) => g !== groupId)
                : [...current, groupId],
        );
    };

    const handleToggleService = (serviceId: string, checked: boolean) => {
        const currentIds = list?.ids || [];
        const newIds = checked
            ? [...currentIds, serviceId]
            : currentIds.filter((id: string) => id !== serviceId);

        dispatch(
            updateBlockedServices({
                ids: newIds,
                schedule: list?.schedule,
            }),
        );
    };

    const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(e.target.value);
    };

    const handleSearchClear = () => {
        setSearch('');
    };

    const isInitialLoading = !allServices && (processing || processingAll);
    const isDisabled = processingSet;

    if (isInitialLoading) {
        return null;
    }

    return (
        <div className={theme.layout.container}>
            <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
            <div className={s.header}>
                <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>{intl.getMessage('blocked_services')}</h1>
                <p className={s.description}>{intl.getMessage('blocked_services_desc')}</p>
            </div>

            <Link to={RoutePath.InactivitySchedule} className={s.navItem}>
                <div className={s.navItemContent}>
                    <div className={s.navItemTitle}>
                        {intl.getMessage('inactivity_schedule')}
                    </div>
                    <div className={s.navItemDesc}>
                        {intl.getMessage('inactivity_schedule_desc')}
                    </div>
                </div>
                <Icon icon="arrow" />
            </Link>

            <div className={s.search}>
                <Input
                    id="blocked-services-search"
                    type="text"
                    value={search}
                    onChange={handleSearchChange}
                    placeholder={intl.getMessage('search_placeholder')}
                    prefixIcon={<Icon icon="search" />}
                    suffixIcon={search ? (
                        <button
                            type="button"
                            onClick={handleSearchClear}
                            className={s.clearButton}
                            aria-label={intl.getMessage('clear_btn')}
                        >
                            <Icon icon="cross" />
                        </button>
                    ) : undefined}
                />
            </div>

            <GroupFilter
                groups={allGroups || []}
                activeGroups={groupFilter}
                onToggleGroup={handleToggleGroup}
            />

            <div className={s.servicesList}>
                {filteredServices.length === 0 ? (
                    <NothingFound />
                ) : (
                    filteredServices.map((service: WebService) => (
                        <ServiceRow
                            key={service.id}
                            id={service.id}
                            name={service.name}
                            iconSvg={service.icon_svg}
                            checked={blockedSet.has(service.id)}
                            disabled={isDisabled}
                            onChange={handleToggleService}
                        />
                    ))
                )}
            </div>
            </div>
        </div>
    );
};
