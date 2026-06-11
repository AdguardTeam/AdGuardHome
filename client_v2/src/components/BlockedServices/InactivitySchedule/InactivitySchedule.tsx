import React, { useEffect, useMemo, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';
import { getAllTimezones, Timezone } from 'countries-and-timezones';
import intl from 'panel/common/intl';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Select } from 'panel/common/controls/Select';
import { RootState } from 'panel/initialState';
import { updateClientFormField } from 'panel/actions/clientForm';
import { getBlockedServices, updateBlockedServices } from 'panel/actions/services';
import theme from 'panel/lib/theme';

import { RoutePath } from 'panel/components/Routes/Paths';
import { buildClientBreadcrumbs } from 'panel/helpers/buildClientBreadcrumbs';
import { ScheduleRow } from './ScheduleRow';
import { ScheduleModal } from './ScheduleModal';
import { DayKey, DAYS_OF_WEEK, ScheduleData, ScheduleDayData, getLocalTimezone } from './helpers';
import { getDayName } from './getDayName';
import s from './InactivitySchedule.module.pcss';

const timezones = getAllTimezones();

const TIMEZONE_OPTIONS = [
    { label: `Local (${getLocalTimezone()})`, value: 'Local' },
    ...Object.entries(timezones).map(([tz, data]: [string, Timezone]) => ({
        label: `${tz} (GMT${data.utcOffsetStr})`,
        value: tz,
    })),
];

type Props = {
    clientScope?: boolean;
};

export const InactivitySchedule = ({ clientScope }: Props) => {
    const dispatch = useDispatch();
    const services = useSelector((state: RootState) => state.services);
    const clientForm = useSelector((state: RootState) => state.clientForm);
    const { list, processing } = services;

    const [modalVisible, setModalVisible] = useState(false);
    const [editDay, setEditDay] = useState<DayKey | undefined>();
    const [deleteDay, setDeleteDay] = useState<DayKey | null>(null);

    useEffect(() => {
        if (!clientScope && !list?.ids) {
            dispatch(getBlockedServices());
        }
    }, [dispatch, clientScope, list]);

    const schedule: ScheduleData | undefined = clientScope
        ? (clientForm.blocked_services_schedule as unknown as ScheduleData)
        : list?.schedule;
    const currentTimezone = schedule?.time_zone || getLocalTimezone();

    const timezoneValue = useMemo(() => {
        return (
            TIMEZONE_OPTIONS.find((opt) => opt.value === currentTimezone) || {
                label: currentTimezone,
                value: currentTimezone,
            }
        );
    }, [currentTimezone]);

    const handleTimezoneChange = (option: { value: string }) => {
        if (!option) {
            return;
        }
        const newSchedule = clientScope
            ? { ...clientForm.blocked_services_schedule, time_zone: option.value }
            : { ...schedule, time_zone: option.value };
        if (clientScope) {
            dispatch(
                updateClientFormField({ field: 'blocked_services_schedule', value: newSchedule }),
            );
        } else {
            dispatch(updateBlockedServices({ ids: list?.ids || [], schedule: newSchedule }));
        }
    };

    const handleAdd = (day: DayKey) => {
        setEditDay(day);
        setModalVisible(true);
    };

    const handleEdit = (day: DayKey) => {
        setEditDay(day);
        setModalVisible(true);
    };

    const handleDelete = (day: DayKey) => {
        setDeleteDay(day);
    };

    const confirmDelete = () => {
        if (!deleteDay) {
            return;
        }
        const newSchedule: Record<string, unknown> = { time_zone: currentTimezone };
        DAYS_OF_WEEK.forEach((d) => {
            if (d !== deleteDay && schedule?.[d]) {
                newSchedule[d] = schedule[d];
            }
        });
        if (clientScope) {
            dispatch(
                updateClientFormField({ field: 'blocked_services_schedule', value: newSchedule }),
            );
        } else {
            dispatch(updateBlockedServices({ ids: list?.ids || [], schedule: newSchedule }));
        }
        setDeleteDay(null);
    };

    const handleModalSave = (day: DayKey, start: number, end: number) => {
        const newSchedule: Record<string, unknown> = { time_zone: currentTimezone };
        DAYS_OF_WEEK.forEach((d) => {
            if (schedule?.[d]) {
                newSchedule[d] = schedule[d];
            }
        });
        newSchedule[day] = { start, end };
        if (clientScope) {
            dispatch(
                updateClientFormField({ field: 'blocked_services_schedule', value: newSchedule }),
            );
        } else {
            dispatch(updateBlockedServices({ ids: list?.ids || [], schedule: newSchedule }));
        }
        setModalVisible(false);
        setEditDay(undefined);
    };

    const handleModalClose = () => {
        setModalVisible(false);
        setEditDay(undefined);
    };

    const parentLinks = clientScope
        ? buildClientBreadcrumbs(clientForm, [
              clientForm.mode === 'edit'
                  ? {
                        path: RoutePath.ClientsEditBlockedServices,
                        title: intl.getMessage('blocked_services'),
                        props: { clientName: encodeURIComponent(clientForm.originalName) },
                    }
                  : {
                        path: RoutePath.ClientsBlockedServices,
                        title: intl.getMessage('blocked_services'),
                    },
          ])
        : [
              {
                  path: RoutePath.BlockedServices,
                  title: intl.getMessage('blocked_services'),
              },
          ];

    const scheduleContent = (
        <>
            <div className={cn(s.timezoneWrapper, clientScope && s.timezoneWrapperClientScope)}>
                <div className={s.timezoneLabel}>
                    {intl.getMessage('inactivity_schedule_timezone')}
                </div>
                <Select
                    options={TIMEZONE_OPTIONS}
                    value={timezoneValue}
                    onChange={handleTimezoneChange}
                    size="responsive"
                    height="big"
                    isDisabled={processing}
                    isSearchable
                />
            </div>

            <div className={s.scheduleList}>
                {DAYS_OF_WEEK.map((day) => (
                    <ScheduleRow
                        key={day}
                        day={day}
                        data={schedule?.[day] as ScheduleDayData | undefined}
                        onEdit={handleEdit}
                        onDelete={handleDelete}
                        onAdd={handleAdd}
                    />
                ))}
            </div>

            {modalVisible && (
                <ScheduleModal
                    visible={modalVisible}
                    currentDay={editDay}
                    currentData={
                        editDay
                            ? (schedule?.[editDay] as ScheduleDayData | undefined)
                            : undefined
                    }
                    onClose={handleModalClose}
                    onSave={handleModalSave}
                />
            )}

            {deleteDay && (
                <ConfirmDialog
                    title={intl.getMessage('inactivity_schedule_delete')}
                    text={intl.getMessage('inactivity_schedule_delete_desc', {
                        value: getDayName(deleteDay),
                    })}
                    buttonText={intl.getMessage('delete_btn')}
                    cancelText={intl.getMessage('cancel_btn')}
                    onConfirm={confirmDelete}
                    onClose={() => setDeleteDay(null)}
                    buttonVariant="danger"
                />
            )}
        </>
    );

    if (clientScope) {
        return scheduleContent;
    }

    return (
        <div className={cn(theme.layout.container, s.container)}>
            <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <div className={s.breadcrumbs}>
                    <Breadcrumbs
                        parentLinks={parentLinks}
                        currentTitle={intl.getMessage('inactivity_schedule')}
                    />
                </div>

                <h1
                    className={cn(
                        theme.layout.title,
                        theme.title.h4,
                        theme.title.h3_tablet,
                        s.title,
                    )}
                >
                    {intl.getMessage('inactivity_schedule')}
                </h1>

                {scheduleContent}
            </div>
        </div>
    );
};
