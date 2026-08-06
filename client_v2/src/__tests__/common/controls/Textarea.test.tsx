import { render } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';
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
});
