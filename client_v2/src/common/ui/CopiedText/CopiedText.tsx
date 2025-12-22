import React, { useState, useCallback, useEffect } from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from '../Icon';

import s from './CopiedText.module.pcss';

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
    const [isCopied, setIsCopied] = useState(false);

    const handleCopy = useCallback(async () => {
        try {
            await navigator.clipboard.writeText(text);
            setIsCopied(true);
            onCopy?.(text);
        } catch (error) {
            console.error('Failed to copy text:', error);
        }
    }, [text, onCopy]);

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
            <div className={cn(s.copyTooltip, { [s.visible]: isCopied })}>
                {intl.getMessage('copied')}
            </div>
        </div>
    );
};
