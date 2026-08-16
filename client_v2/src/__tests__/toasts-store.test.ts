import { describe, it, expect, beforeEach } from 'vitest';
import {
    addErrorToast,
    addSuccessToast,
    addNoticeToast,
    removeToast,
    toastsState,
} from '../stores/toasts';

describe('toasts store', () => {
    beforeEach(() => {
        // Clear all notices between tests.
        toastsState.notices.forEach((n: any) => removeToast(n.id));
    });

    it('addNoticeToast stores the message', () => {
        addNoticeToast('update_failed');
        const last = toastsState.notices[toastsState.notices.length - 1];
        expect(last.message).toBe('update_failed');
        expect(last.type).toBe('notice');
    });

    it('addErrorToast preserves options and action', () => {
        const action = { text: 'retry', callback: () => {} };
        addErrorToast({ error: 'boom', options: { x: 1 }, action });
        const last = toastsState.notices[toastsState.notices.length - 1];
        expect(last.message).toBe('boom');
        expect(last.options).toEqual({ x: 1 });
        expect(last.action).toEqual(action);
        expect(last.type).toBe('error');
    });

    it('addSuccessToast preserves code on object payload', () => {
        addSuccessToast({ message: 'notify_updated', code: 'notify_updated' });
        const last = toastsState.notices[toastsState.notices.length - 1];
        expect(last.code).toBe('notify_updated');
    });

    it('addSuccessToast accepts a bare string', () => {
        addSuccessToast('config_successfully_saved');
        const last = toastsState.notices[toastsState.notices.length - 1];
        expect(last.message).toBe('config_successfully_saved');
    });
});
