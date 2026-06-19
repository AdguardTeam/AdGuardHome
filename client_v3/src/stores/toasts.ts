import { createStore, reconcile } from 'solid-js/store';
import { nanoid } from 'nanoid';

type ToastNotice = {
    id: string;
    message: any;
    type: 'error' | 'success' | 'notice';
    actionLabel?: string;
    undoId?: string;
};

type ToastsState = {
    notices: ToastNotice[];
};

const initialState: ToastsState = {
    notices: [],
};

const [state, setState] = createStore<ToastsState>(initialState);

type UndoCallback = () => void | Promise<void>;

const undoRegistry = new Map<string, UndoCallback>();

export const getUndoCallback = (id: string): UndoCallback | undefined => undoRegistry.get(id);

export const clearUndoCallback = (id: string): void => {
    undoRegistry.delete(id);
};

export const createUndoToast = (
    message: any,
    actionLabel: string,
    onUndo: UndoCallback,
): { message: any; actionLabel: string; undoId: string } => {
    const undoId = nanoid();
    undoRegistry.set(undoId, onUndo);
    return { message, actionLabel, undoId };
};

export const addErrorToast = (payload: { error: any }) => {
    const { error } = payload;
    const message = error instanceof Error ? error.message : String(error);
    console.error(message); // eslint-disable-line no-console
    setState('notices', (prev) => [...prev, { id: nanoid(), message, type: 'error' as const }]);
};

export const addSuccessToast = (message: any) => {
    const notice: ToastNotice = {
        id: nanoid(),
        type: 'success',
        message: typeof message === 'string' ? message : message?.message || message,
    };
    if (typeof message === 'object' && message !== null) {
        notice.actionLabel = message.actionLabel;
        notice.undoId = message.undoId;
    }
    setState('notices', (prev) => [...prev, notice]);
};

export const addNoticeToast = (message: any) => {
    setState('notices', (prev) => [...prev, { id: nanoid(), message, type: 'notice' as const }]);
};

export const removeToast = (id: string) => {
    setState('notices', (prev) => prev.filter((n) => n.id !== id));
};

export const toastsState = state;
