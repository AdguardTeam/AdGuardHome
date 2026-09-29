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

    it('collapses a repeated identical error into one notice', () => {
        addErrorToast({ error: 'boom' });
        addErrorToast({ error: 'boom' });

        expect(toastsState.notices.filter((n) => n.message === 'boom')).toHaveLength(1);
    });

    it('keeps notices with different messages apart', () => {
        addErrorToast({ error: 'boom' });
        addErrorToast({ error: 'bang' });

        expect(toastsState.notices).toHaveLength(2);
    });

    it('does not collapse an error that carries an action', () => {
        const action = { text: 'retry', callback: () => {} };
        addErrorToast({ error: 'boom', action });
        addErrorToast({ error: 'boom' });

        expect(toastsState.notices).toHaveLength(2);
    });
});
