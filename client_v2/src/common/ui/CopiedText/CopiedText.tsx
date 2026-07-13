import { createSignal, createEffect, onCleanup } from 'solid-js';
import cn from 'clsx';
import copy from 'copy-to-clipboard';
import intl from 'panel/common/intl';
import { addSuccessToast } from 'panel/stores/toasts';
import { Icon } from '../Icon';

import s from './CopiedText.module.pcss';

export type CopiedTextProps = {
    text: string;
    class?: string;
    onCopy?: (text: string) => void;
};

export const CopiedText = (props: CopiedTextProps) => {
    const [isCopied, setIsCopied] = createSignal(false);
    let timer: ReturnType<typeof setTimeout> | undefined;

    const handleCopy = async () => {
        try {
            const ok = copy(props.text);

            if (!ok) {
                throw new Error('Failed to copy text');
            }

            setIsCopied(true);
            addSuccessToast(intl.getMessage('copied'));
            props.onCopy?.(props.text);
        } catch (error) {
            console.error('Failed to copy text:', error);
        }
    };

    const resetTimer = () => {
        if (timer) {
            clearTimeout(timer);
        }
        if (isCopied()) {
            timer = setTimeout(() => {
                setIsCopied(false);
            }, 2000);
        }
    };

    // Reset timer when isCopied changes
    createEffect(() => {
        if (isCopied()) resetTimer();
    });

    onCleanup(() => {
        if (timer) {
            clearTimeout(timer);
        }
    });

    return (
        <div
            class={cn(s.container, props.class)}
            onClick={handleCopy}
            role="button"
            aria-label={isCopied() ? intl.getMessage('copied') : intl.getMessage('copy')}
        >
            <span class={s.text}>{props.text}</span>
            <Icon icon="copy" class={cn(s.icon, { [s.copied]: isCopied() })} />
        </div>
    );
};
