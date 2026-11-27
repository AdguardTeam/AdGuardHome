import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import {
    blockedServicesSchedule,
    blockedServicesAll,
    blockedServicesScheduleUpdate,
} from 'panel/api/generated';
import { addErrorToast, addSuccessToast, createUndoToast } from './toasts';
import intl from 'panel/common/intl';
import type { BlockedServicesSchedule } from 'panel/api/model/blockedServicesSchedule';
import type { BlockedService } from 'panel/api/model/blockedService';
import type { ServiceGroup } from 'panel/api/model/serviceGroup';

type ServicesState = {
    processing: boolean;
    /** Whether the blocked-services schedule request has settled at least once. */
    initialized: boolean;
    processingAll: boolean;
    /** Whether the all-blocked-services request has settled at least once. */
    allInitialized: boolean;
    processingSet: boolean;
    /** The service whose switch triggered the pending whole-list update. */
    processingServiceId: string | null;
    list: BlockedServicesSchedule;
    allServices: BlockedService[];
    allGroups: ServiceGroup[];
};

const initialState: ServicesState = {
    processing: true,
    initialized: false,
    processingAll: true,
    allInitialized: false,
    processingSet: false,
    processingServiceId: null,
    list: {},
    allServices: [],
    allGroups: [],
};

const [state, setState] = createStore<ServicesState>(initialState);

// Whole-list updates and reads race each other: a read that started before a
// PUT must not restore the pre-update ids over the accepted payload.
let latestServicesGet = 0;

const beginServicesGet = () => {
    latestServicesGet += 1;
    return latestServicesGet;
};

const isLatestServicesGet = (id: number) => id === latestServicesGet;

export const getBlockedServices = async () => {
    const requestId = beginServicesGet();
    setState('processing', true);
    try {
        const data = await blockedServicesSchedule();
        if (!isLatestServicesGet(requestId)) {
            return;
        }
        setState({ list: data, processing: false, initialized: true });
    } catch (error) {
        if (!isLatestServicesGet(requestId)) {
            return;
        }
        addErrorToast({ error });
        setState({ processing: false, initialized: true });
    }
};

export const getAllBlockedServices = async () => {
    setState('processingAll', true);
    try {
        const data = await blockedServicesAll();
        setState({
            allServices: data.blocked_services || [],
            allGroups: data.groups || [],
            processingAll: false,
            allInitialized: true,
        });
    } catch (error) {
        addErrorToast({ error });
        setState({ processingAll: false, allInitialized: true });
    }
};

export const updateBlockedServices = async (
    values: BlockedServicesSchedule,
    serviceId?: string,
): Promise<boolean> => {
    setState({ processingSet: true, processingServiceId: serviceId ?? null });
    const updateId = beginServicesGet();
    try {
        await blockedServicesScheduleUpdate(values);
        // The accepted payload is authoritative until the follow-up read lands;
        // a failed read must not re-enable editing against a stale list.
        setState('list', values);
        await getBlockedServices();
        return true;
    } catch (error) {
        addErrorToast({ error });
        if (isLatestServicesGet(updateId)) {
            setState('processing', false);
        }
        return false;
    } finally {
        setState({ processingSet: false, processingServiceId: null });
    }
};

export const allowBlockedService = async (serviceId: string): Promise<boolean> => {
    let list = untrack(() => state.list);
    if (!Array.isArray(list?.ids)) {
        await getBlockedServices();
        list = untrack(() => state.list);
    }
    const currentIds = Array.isArray(list?.ids) ? list.ids : [];
    if (!currentIds.includes(serviceId)) return true;
    const serviceName =
        state.allServices.find((s) => s.id === serviceId)?.name || serviceId;
    const didUpdate = await updateBlockedServices(
        {
            ids: currentIds.filter((id: string) => id !== serviceId),
            schedule: list?.schedule,
        },
        serviceId,
    );
    if (!didUpdate) return false;
    addSuccessToast(
        createUndoToast(
            intl.getMessage('user_rules_service_allowed', { value: serviceName }),
            intl.getMessage('notify_undo'),
            async () => {
                await updateBlockedServices(
                    {
                        ids: [...currentIds, serviceId],
                        schedule: list?.schedule,
                    },
                    serviceId,
                );
            },
        ),
    );
    return true;
};

export const servicesState = untrack(() => state);
