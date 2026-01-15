import React, { useState, useCallback, useEffect } from 'react';
import { useDispatch } from 'react-redux';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { addSuccessToast } from 'panel/actions/toasts';
import { Icon } from '../Icon';


import s from './CopiedText.module.pcss';

const copyTextToClipboard = async (text: string) => {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
        return;
    }

    if (typeof document === 'undefined') {
        throw new Error('Clipboard is not available');
    }

    const el = document.createElement('textarea');
    el.value = text;
    el.setAttribute('readonly', '');
    el.style.position = 'fixed';
    el.style.top = '0';
    el.style.left = '0';
    el.style.opacity = '0';

    document.body.appendChild(el);
    el.focus();
    el.select();

    const ok = document.execCommand('copy');
    document.body.removeChild(el);

    if (!ok) {
        throw new Error('Failed to copy text');
    }
};

export type CopiedTextProps = {
    text: string;
    className?: string;
    onCopy?: (text: string) => void;
}

export const CopiedText = ({
    text,
    className,
    onCopy,
}: CopiedTextProps) => {
    const dispatch = useDispatch();
    const [isCopied, setIsCopied] = useState(false);

    const handleCopy = useCallback(async () => {
        try {
            await copyTextToClipboard(text);
            setIsCopied(true);
            dispatch(addSuccessToast(intl.getMessage('copied')));
            onCopy?.(text);
        } catch (error) {
            console.error('Failed to copy text:', error);
        }
    }, [dispatch, text, onCopy]);

    useEffect(() => {
        let timer: NodeJS.Timeout;

        if (isCopied) {
            timer = setTimeout(() => {
                setIsCopied(false);
            }, 2000);
        }

        return () => {
            if (timer) {
                clearTimeout(timer);
            }
        };
    }, [isCopied]);

    return (
        <div
            className={cn(s.container, className)}
            onClick={handleCopy}
            role="button"
            aria-label={isCopied ? intl.getMessage('copied') : intl.getMessage('copy')}
        >
            <span
                className={s.text}
            >
                {text}
            </span>
            <Icon
                icon="copy"
                className={cn(s.icon, { [s.copied]: isCopied })}
            />
        </div>
    );
};
