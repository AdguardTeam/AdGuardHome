import { fireEvent, render } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';

import { RewritesTable } from 'panel/components/FilterLists/blocks/RewritesTable';
import type { Rewrite } from 'panel/components/FilterLists/DNSRewrites';

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
    },
}));

const renderTable = (answers: string[]) => {
    const answerSet = new Set(answers);
    const list: Rewrite[] = answers.map((answer, index) => ({
        answer,
        domain: `${index}.example`,
        enabled: true,
    }));
    const view = render(() => (
        <RewritesTable
            list={list}
            processing={false}
            processingAdd={false}
            processingDelete={false}
            processingUpdate={false}
            addRewritesList={vi.fn()}
            deleteRewrite={vi.fn()}
            editRewrite={vi.fn()}
            toggleRewrite={vi.fn()}
        />
    ));

    const answersInDocumentOrder = () =>
        Array.from(view.container.querySelectorAll('span'))
            .map((element) => element.textContent || '')
            .filter((text) => answerSet.has(text));

    return {
        ...view,
        answersInDocumentOrder,
    };
};

describe('RewritesTable', () => {
    it('sorts IPv4 rewrite results numerically in both directions', () => {
        const view = renderTable(['172.18.0.10', '172.18.0.2', '172.18.0.5']);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual(['172.18.0.2', '172.18.0.5', '172.18.0.10']);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual(['172.18.0.10', '172.18.0.5', '172.18.0.2']);
    });

    it('sorts IPv4 before IPv6 and orders both families numerically', () => {
        const view = renderTable(['2001:db8::10', '172.18.0.10', '2001:db8::2', '172.18.0.2']);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual([
            '172.18.0.2',
            '172.18.0.10',
            '2001:db8::2',
            '2001:db8::10',
        ]);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual([
            '2001:db8::10',
            '2001:db8::2',
            '172.18.0.10',
            '172.18.0.2',
        ]);
    });

    it('sorts IP addresses before text in ascending order', () => {
        const view = renderTable(['beta.example', '172.18.0.10', 'alpha.example', '172.18.0.2']);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual([
            '172.18.0.2',
            '172.18.0.10',
            'alpha.example',
            'beta.example',
        ]);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual([
            'beta.example',
            'alpha.example',
            '172.18.0.10',
            '172.18.0.2',
        ]);
    });

    it('sorts text case-insensitively and preserves equal values order', () => {
        const view = renderTable([
            'Zeta.Example',
            'Beta.example',
            'alpha.example',
            'beta.EXAMPLE',
        ]);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual([
            'alpha.example',
            'Beta.example',
            'beta.EXAMPLE',
            'Zeta.Example',
        ]);
    });

    it('treats noncanonical IPv4-like hostnames as text', () => {
        const view = renderTable(['250.0.0.1', '0x7f000001', 'alpha.example']);

        fireEvent.click(view.getByText('result'));

        expect(view.answersInDocumentOrder()).toEqual(['250.0.0.1', '0x7f000001', 'alpha.example']);
    });
});
