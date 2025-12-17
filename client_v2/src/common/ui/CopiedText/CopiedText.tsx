import React, { useState, useCallback } from 'react';
import cn from 'clsx';
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

    return (
        <div className={cn(s.container, className)}>
            <span
                className={s.text}
            >
                {text}
            </span>
            <Icon
                icon="copy"
                className={cn(s.icon, { [s.copied]: isCopied })}
                onClick={handleCopy}
            />
        </div>
    );
};
