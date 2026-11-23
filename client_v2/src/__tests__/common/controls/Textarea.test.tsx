import { render } from '@solidjs/testing-library';
import { describe, expect, it } from 'vitest';
import { Textarea } from 'panel/common/controls/Textarea';
import { COMMENT_LINE_TOKENS } from 'panel/helpers/constants';

describe('Textarea — comment highlight', () => {
    // Use template literals with real newlines — JSX string attributes
    // treat \n as a literal backslash-n in some transform pipelines.
    const multilineValue = `# comment
||example.com^`;

    const commentOnlyValue = `! comment
||example.com^`;

    const indentedCommentValue = '   # indented comment';

    it('renders overlay with all text when highlightComments is enabled', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const overlay = container.querySelector('[aria-hidden="true"]');
        expect(overlay).toBeInTheDocument();
        expect(overlay?.textContent).toContain('# comment');
        expect(overlay?.textContent).toContain('||example.com^');
    });

    it('does not render overlay when highlightComments is disabled', () => {
        const { container } = render(() => <Textarea value={multilineValue} />);

        const overlay = container.querySelector('[aria-hidden="true"]');
        expect(overlay).not.toBeInTheDocument();
    });

    it('applies the shared scrollbar styles to the native textarea', () => {
        const { container } = render(() => <Textarea value={multilineValue} />);

        expect(container.querySelector('textarea')?.className).toContain('scrollable');
    });

    it('applies the shared scrollbar styles to the highlighted scroll container', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const scrollArea = container.querySelector('textarea')?.closest('[class*="scrollArea"]');
        expect(scrollArea?.className).toContain('scrollable');
    });

    it('applies commentLine class to lines starting with #', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const overlay = container.querySelector('[aria-hidden="true"]');
        const spans = overlay?.querySelectorAll('span');
        expect(spans).toHaveLength(2);
        expect(spans?.[0].className).toContain('commentLine');
        expect(spans?.[1].className).toContain('overlayNonComment');
    });

    it('detects ! as comment prefix when configured', () => {
        const { container } = render(() => (
            <Textarea
                value={commentOnlyValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const overlay = container.querySelector('[aria-hidden="true"]');
        const spans = overlay?.querySelectorAll('span');
        expect(spans?.[0].className).toContain('commentLine');
    });

    it('detects comment with leading whitespace', () => {
        const { container } = render(() => (
            <Textarea
                value={indentedCommentValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const overlay = container.querySelector('[aria-hidden="true"]');
        const spans = overlay?.querySelectorAll('span');
        expect(spans?.[0].className).toContain('commentLine');
    });

    it('groups consecutive comment lines into a single span', () => {
        const value = `# first\n# second\n# third`;
        const { container } = render(() => (
            <Textarea
                value={value}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const overlay = container.querySelector('[aria-hidden="true"]');
        const spans = overlay?.querySelectorAll('span');
        expect(spans).toHaveLength(1);
        expect(spans?.[0].className).toContain('commentLine');
        expect(spans?.[0].textContent).toBe(value);
    });

    it('groups alternating comment and rule blocks', () => {
        const value = `# first\n# second\n||a.com^\n||b.com^\n# third`;
        const { container } = render(() => (
            <Textarea
                value={value}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const overlay = container.querySelector('[aria-hidden="true"]');
        const spans = overlay?.querySelectorAll('span');
        expect(spans).toHaveLength(3);
        expect(spans?.[0].textContent).toBe('# first\n# second');
        expect(spans?.[1].textContent).toBe('||a.com^\n||b.com^');
        expect(spans?.[2].textContent).toBe('# third');
    });

    it('applies transparentText class to textarea when highlightComments enabled', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        const textarea = container.querySelector('textarea');
        expect(textarea?.className).toContain('transparentText');
    });

    it('does not apply transparentText when highlightComments disabled', () => {
        const { container } = render(() => <Textarea value={multilineValue} />);

        const textarea = container.querySelector('textarea');
        expect(textarea?.className).not.toContain('transparentText');
    });

    it('renders the shared scroll container in highlighting mode', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
            />
        ));

        // The overlay and textarea should share a parent
        const overlay = container.querySelector('[aria-hidden="true"]');
        const textarea = container.querySelector('textarea');
        expect(overlay?.parentElement).toBe(textarea?.parentElement);
    });

    it('bounds the scroll container to the size when rows are not provided', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
                size="large"
            />
        ));

        const scrollArea = container.querySelector('textarea')?.closest('[class*="scrollArea"]');
        expect(scrollArea?.className).toContain('bounded');
    });

    it('does not bound the scroll container when rows are provided', () => {
        const { container } = render(() => (
            <Textarea
                value={multilineValue}
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
                size="large"
                rows={12}
            />
        ));

        const scrollArea = container.querySelector('textarea')?.closest('[class*="scrollArea"]');
        expect(scrollArea?.className).not.toContain('bounded');
    });
});

describe('Textarea — onCard', () => {
    it('applies the onCard class to the field when set', () => {
        const { container } = render(() => <Textarea value="text" onCard />);

        expect(container.querySelector('textarea')?.className).toContain('onCard');
    });

    it('does not apply the onCard class by default', () => {
        const { container } = render(() => <Textarea value="text" />);

        expect(container.querySelector('textarea')?.className).not.toContain('onCard');
    });

    it('applies the onCard class to the scroll container in highlighting mode', () => {
        const { container } = render(() => (
            <Textarea
                value="# comment"
                highlightComments
                commentPrefixes={COMMENT_LINE_TOKENS}
                onCard
            />
        ));

        // The visible surface in highlighting mode is the scroll container,
        // not the transparent textarea layered on top of it.
        const textarea = container.querySelector('textarea');
        const scrollArea = textarea?.closest('[class*="scrollArea"]');
        expect(scrollArea?.className).toContain('onCard');
        expect(textarea?.className).not.toContain('onCard');
    });
});
