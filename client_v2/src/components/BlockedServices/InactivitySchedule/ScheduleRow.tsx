import { createSignal, Show } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';

import { DayKey, ScheduleDayData, formatTimePeriod, isFullDay } from './helpers';
import { getDayName } from './getDayName';
import s from './InactivitySchedule.module.pcss';

type Props = {
    day: DayKey;
    data: ScheduleDayData | undefined;
    onEdit: (day: DayKey) => void;
    onDelete: (day: DayKey) => void;
    onAdd: (day: DayKey) => void;
};

export const ScheduleRow = (props: Props) => {
    const dayName = () => getDayName(props.day);
    const isConfigured = () => !!props.data;
    const [dropdownOpen, setDropdownOpen] = createSignal(false);

    const handleEditClick = () => {
        setDropdownOpen(false);
        props.onEdit(props.day);
    };

    const handleDeleteClick = () => {
        setDropdownOpen(false);
        props.onDelete(props.day);
    };

    const getTimeDisplay = () => {
        if (!props.data) {
            return (
                <>
                    <span class={s.noScheduleTextDesktop}>
                        {intl.getMessage('inactivity_schedule_no_schedules')}
                    </span>
                    <span class={s.noScheduleTextMobile}>–</span>
                </>
            );
        }

        if (isFullDay(props.data.start, props.data.end)) {
            return intl.getMessage('inactivity_schedule_all_day');
        }

        return formatTimePeriod(props.data.start, props.data.end);
    };

    return (
        <div class={s.scheduleRow} data-testid={`schedule-row-${props.day}`}>
            <div class={s.dayName} data-testid={`schedule-row-${props.day}-day`}>{dayName()}</div>
            <div class={s.timeDisplay} data-testid={`schedule-row-${props.day}-time`}>{getTimeDisplay()}</div>
            <Show
                when={isConfigured()}
                fallback={
                    <>
                        <div class={cn(s.actions, s.actionsDesktop)} />
                        <div class={s.actions}>
                            <button
                                type="button"
                                class={cn(s.actionButton, s.actionButtonAdd)}
                                onClick={() => props.onAdd(props.day)}
                                aria-label={intl.getMessage('inactivity_schedule_add')}
                                data-testid={`schedule-row-${props.day}-add`}
                            >
                                <Icon icon="plus" />
                            </button>
                        </div>
                    </>
                }
            >
                <div class={cn(s.actions, s.actionsDesktop)}>
                    <button
                        type="button"
                        class={cn(s.actionButton, s.actionButtonDelete)}
                        onClick={() => props.onDelete(props.day)}
                        aria-label={intl.getMessage('delete_table_action')}
                        data-testid={`schedule-row-${props.day}-delete`}
                    >
                        <Icon icon="delete" />
                    </button>
                </div>
                <div class={cn(s.actions, s.actionsDesktop)}>
                    <button
                        type="button"
                        class={s.actionButton}
                        onClick={() => props.onEdit(props.day)}
                        aria-label={intl.getMessage('inactivity_schedule_edit')}
                        data-testid={`schedule-row-${props.day}-edit`}
                    >
                        <Icon icon="edit" />
                    </button>
                </div>
                <div class={cn(s.actions, s.actionsMobile)}>
                    <Dropdown
                        trigger="click"
                        position="bottomRight"
                        noIcon
                        open={dropdownOpen()}
                        onOpenChange={setDropdownOpen}
                        menu={
                            <div class={theme.dropdown.menu}>
                                <button
                                    type="button"
                                    class={theme.dropdown.item}
                                    onClick={handleEditClick}
                                >
                                    {intl.getMessage('inactivity_schedule_edit')}
                                </button>
                                <button
                                    type="button"
                                    class={cn(theme.dropdown.item, s.dropdownItemDanger)}
                                    onClick={handleDeleteClick}
                                >
                                    {intl.getMessage('inactivity_schedule_delete')}
                                </button>
                            </div>
                        }
                    >
                        <div class={s.actionButton}>
                            <Icon icon="bullets" />
                        </div>
                    </Dropdown>
                </div>
            </Show>
        </div>
    );
};
