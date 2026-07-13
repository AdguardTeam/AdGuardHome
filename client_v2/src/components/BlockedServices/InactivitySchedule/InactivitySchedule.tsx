import { createSignal, createMemo, onMount, Show, For } from 'solid-js';
import { getAllTimezones, type Timezone } from 'countries-and-timezones';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { PageLoader } from 'panel/common/ui/Loader';
import { Select } from 'panel/common/controls/Select';
import { updateClientFormField, clientFormState } from 'panel/stores/clientForm';
import { getBlockedServices, updateBlockedServices, servicesState } from 'panel/stores/services';
import theme from 'panel/lib/theme';

import { RoutePath } from 'panel/components/Routes/Paths';
import { buildClientBreadcrumbs } from 'panel/helpers/buildClientBreadcrumbs';
import { ScheduleRow } from './ScheduleRow';
import { ScheduleModal } from './ScheduleModal';
import { type DayKey, DAYS_OF_WEEK, type ScheduleData, type ScheduleDayData } from './helpers';
import { getDayName } from './getDayName';
import s from './InactivitySchedule.module.pcss';

const timezones = getAllTimezones();

const TIMEZONE_OPTIONS = [
    { label: 'Local', value: 'Local' },
    { label: 'UTC (GMT+00:00)', value: 'UTC' },
    ...Object.entries(timezones).map(([tz, data]: [string, Timezone]) => ({
        label: `${tz} (GMT${data.utcOffsetStr})`,
        value: tz,
    })),
];

type Props = {
    clientScope?: boolean;
};

export const InactivitySchedule = (props: Props) => {
    const [modalVisible, setModalVisible] = createSignal(false);
    const [editDay, setEditDay] = createSignal<DayKey | undefined>();
    const [deleteDay, setDeleteDay] = createSignal<DayKey | null>(null);

    onMount(() => {
        if (!props.clientScope && !servicesState.list?.ids) {
            getBlockedServices();
        }
    });

    const schedule = createMemo<ScheduleData | undefined>(() => {
        return props.clientScope
            ? (clientFormState.blocked_services_schedule as unknown as ScheduleData)
            : servicesState.list?.schedule;
    });

    const currentTimezone = () => schedule()?.time_zone;

    const timezoneValue = createMemo(() => {
        const tz = currentTimezone();
        if (!tz) return undefined;
        return (
            TIMEZONE_OPTIONS.find((opt) => opt.value === tz) || {
                label: tz,
                value: tz,
            }
        );
    });

    const isLoading = createMemo(() => {
        if (props.clientScope) return false;
        return servicesState.processing && !servicesState.list?.ids;
    });

    const handleTimezoneChange = (option: { value: string }) => {
        if (!option) {
            return;
        }
        const newSchedule = props.clientScope
            ? { ...(clientFormState.blocked_services_schedule as any), time_zone: option.value }
            : { ...schedule(), time_zone: option.value };
        if (props.clientScope) {
            updateClientFormField('blocked_services_schedule', newSchedule, true);
        } else {
            updateBlockedServices({ ids: servicesState.list?.ids || [], schedule: newSchedule });
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
        const dayToDelete = deleteDay();
        if (!dayToDelete) {
            return;
        }
        const newSchedule: Record<string, unknown> = { time_zone: currentTimezone() };
        DAYS_OF_WEEK.forEach((d) => {
            if (d !== dayToDelete && schedule()?.[d]) {
                newSchedule[d] = schedule()![d];
            }
        });
        if (props.clientScope) {
            updateClientFormField('blocked_services_schedule', newSchedule, true);
        } else {
            updateBlockedServices({ ids: servicesState.list?.ids || [], schedule: newSchedule });
        }
        setDeleteDay(null);
    };

    const handleModalSave = (day: DayKey, start: number, end: number) => {
        const newSchedule: Record<string, unknown> = { time_zone: currentTimezone() };
        DAYS_OF_WEEK.forEach((d) => {
            if (schedule()?.[d]) {
                newSchedule[d] = schedule()![d];
            }
        });
        newSchedule[day] = { start, end };
        if (props.clientScope) {
            updateClientFormField('blocked_services_schedule', newSchedule, true);
        } else {
            updateBlockedServices({ ids: servicesState.list?.ids || [], schedule: newSchedule });
        }
        setModalVisible(false);
        setEditDay(undefined);
    };

    const handleModalClose = () => {
        setModalVisible(false);
        setEditDay(undefined);
    };

    const parentLinks = () =>
        props.clientScope
            ? buildClientBreadcrumbs(clientFormState, [
                  clientFormState.mode === 'edit'
                      ? {
                            path: RoutePath.ClientsEditBlockedServices,
                            title: intl.getMessage('blocked_services'),
                            props: { clientName: encodeURIComponent(clientFormState.originalName) },
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

    return (
        <Show
            when={props.clientScope}
            fallback={
                <Show
                    when={!isLoading()}
                    fallback={
                        <div class={cn(theme.layout.container, s.container)}>
                            <PageLoader />
                        </div>
                    }
                >
                    <div class={cn(theme.layout.container, s.container)}>
                        <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                            <div class={s.breadcrumbs} data-testid="inactivity-schedule-breadcrumbs">
                                <Breadcrumbs
                                    parentLinks={parentLinks()}
                                    currentTitle={intl.getMessage('inactivity_schedule')}
                                />
                            </div>

                            <h1
                                class={cn(
                                    theme.layout.title,
                                    theme.title.h4,
                                    theme.title.h3_tablet,
                                    s.title,
                                )}
                                data-testid="inactivity-schedule-title"
                            >
                                {intl.getMessage('inactivity_schedule')}
                            </h1>

                            <div
                                class={cn(
                                    s.timezoneWrapper,
                                    props.clientScope && s.timezoneWrapperClientScope,
                                )}
                                data-testid="inactivity-schedule-timezone"
                            >
                                <div class={s.timezoneLabel}>
                                    {intl.getMessage('inactivity_schedule_timezone')}
                                </div>
                                <Select
                                    options={TIMEZONE_OPTIONS}
                                    value={timezoneValue()}
                                    onChange={handleTimezoneChange}
                                    size="responsive"
                                    height="big"
                                    isSearchable
                                />
                            </div>

                            <div class={s.scheduleList} data-testid="inactivity-schedule-list">
                                <For each={DAYS_OF_WEEK}>
                                    {(day) => (
                                        <ScheduleRow
                                            day={day}
                                            data={schedule()?.[day] as ScheduleDayData | undefined}
                                            onEdit={handleEdit}
                                            onDelete={handleDelete}
                                            onAdd={handleAdd}
                                        />
                                    )}
                                </For>
                            </div>

                            <Show when={modalVisible()}>
                                <ScheduleModal
                                    visible={modalVisible()}
                                    currentDay={editDay()}
                                    currentData={
                                        editDay()
                                            ? (schedule()?.[editDay()!] as
                                                  | ScheduleDayData
                                                  | undefined)
                                            : undefined
                                    }
                                    onClose={handleModalClose}
                                    onSave={handleModalSave}
                                />
                            </Show>

                            <Show when={deleteDay()}>
                                <ConfirmDialog
                                    title={intl.getMessage('inactivity_schedule_delete')}
                                    text={intl.getMessage('inactivity_schedule_delete_desc', {
                                        value: getDayName(deleteDay()!),
                                    })}
                                    buttonText={intl.getMessage('delete_btn')}
                                    cancelText={intl.getMessage('cancel_btn')}
                                    onConfirm={confirmDelete}
                                    onClose={() => setDeleteDay(null)}
                                    buttonVariant="danger"
                                />
                            </Show>
                        </div>
                    </div>
                </Show>
            }
        >
            {/* clientScope - render inline */}
            <div class={cn(s.timezoneWrapper, s.timezoneWrapperClientScope)} data-testid="inactivity-schedule-timezone">
                <div class={s.timezoneLabel}>{intl.getMessage('inactivity_schedule_timezone')}</div>
                <Select
                    options={TIMEZONE_OPTIONS}
                    value={timezoneValue()}
                    onChange={handleTimezoneChange}
                    size="responsive"
                    height="big"
                    isDisabled={clientFormState.processingSave}
                    isSearchable
                />
            </div>

            <div class={s.scheduleList} data-testid="inactivity-schedule-list">
                <For each={DAYS_OF_WEEK}>
                    {(day) => (
                        <ScheduleRow
                            day={day}
                            data={schedule()?.[day] as ScheduleDayData | undefined}
                            onEdit={handleEdit}
                            onDelete={handleDelete}
                            onAdd={handleAdd}
                        />
                    )}
                </For>
            </div>

            <Show when={modalVisible()}>
                <ScheduleModal
                    visible={modalVisible()}
                    currentDay={editDay()}
                    currentData={
                        editDay()
                            ? (schedule()?.[editDay()!] as ScheduleDayData | undefined)
                            : undefined
                    }
                    onClose={handleModalClose}
                    onSave={handleModalSave}
                />
            </Show>

            <Show when={deleteDay()}>
                <ConfirmDialog
                    title={intl.getMessage('inactivity_schedule_delete')}
                    text={intl.getMessage('inactivity_schedule_delete_desc', {
                        value: getDayName(deleteDay()!),
                    })}
                    buttonText={intl.getMessage('delete_btn')}
                    cancelText={intl.getMessage('cancel_btn')}
                    onConfirm={confirmDelete}
                    onClose={() => setDeleteDay(null)}
                    buttonVariant="danger"
                />
            </Show>
        </Show>
    );
};
