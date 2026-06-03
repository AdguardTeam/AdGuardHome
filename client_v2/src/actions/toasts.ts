import { createAction } from 'redux-actions';
import { nanoid } from 'nanoid';
import type { ReactNode } from 'react';

type UndoCallback = () => void | Promise<void>;

const undoRegistry = new Map<string, UndoCallback>();

/** Retrieve an undo callback by toast ID. Called by the Toast component. */
export const getUndoCallback = (id: string): UndoCallback | undefined =>
    undoRegistry.get(id);

/** Remove an undo callback from the registry after the toast is dismissed. */
export const clearUndoCallback = (id: string): void => {
    undoRegistry.delete(id);
};

type UndoToastPayload = {
    message: ReactNode;
    actionLabel: string;
    undoId: string;
};

/**
 * Create a success toast payload with an undo action.
 * The callback is stored in the registry so the payload stays serializable.
 */
export const createUndoToast = (
    message: ReactNode,
    actionLabel: string,
    onUndo: UndoCallback,
): UndoToastPayload => {
    const undoId = nanoid();
    undoRegistry.set(undoId, onUndo);
    return { message, actionLabel, undoId };
};

export const addErrorToast = createAction('ADD_ERROR_TOAST');
export const addSuccessToast = createAction('ADD_SUCCESS_TOAST');
export const addNoticeToast = createAction('ADD_NOTICE_TOAST');
