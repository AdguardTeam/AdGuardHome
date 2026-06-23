import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    deleteRewrite: vi.fn(),
    getRewritesList: vi.fn(),
    addSuccessToast: vi.fn(),
    addErrorToast: vi.fn(),
    getMessage: vi.fn((key: string, values?: any) => {
        if (values?.key) return `${key} ${values.key}`;
        return key;
    }),
}));

vi.mock('panel/api/Api', () => ({
    apiClient: {
        deleteRewrite: mocks.deleteRewrite,
        getRewritesList: mocks.getRewritesList,
    },
}));
vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
}));
vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: mocks.getMessage,
    },
}));

import { deleteRewrite } from 'panel/stores/rewrites';

describe('deleteRewrite', () => {
    beforeEach(() => vi.clearAllMocks());

    it('shows rewrite_deleted toast with the domain (FR-009)', async () => {
        mocks.deleteRewrite.mockResolvedValue(undefined);
        mocks.getRewritesList.mockResolvedValue([]);
        await deleteRewrite({
            domain: 'example.com',
            answer: '1.1.1.1',
            enabled: true,
        });
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(
            expect.stringContaining('example.com'),
        );
    });

    it('honors showToast:false', async () => {
        mocks.deleteRewrite.mockResolvedValue(undefined);
        mocks.getRewritesList.mockResolvedValue([]);
        await deleteRewrite(
            { domain: 'example.com', answer: '1.1.1.1', enabled: true },
            { showToast: false },
        );
        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
    });
});
