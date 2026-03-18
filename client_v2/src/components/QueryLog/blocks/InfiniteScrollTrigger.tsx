import React, { useCallback, useEffect, useRef } from 'react';
import cn from 'clsx';

import { InlineLoader } from 'panel/common/ui/Loader';

import s from './InfiniteScrollTrigger.module.pcss';

type Props = {
    hasMore: boolean;
    loading: boolean;
    disabled: boolean;
    onLoadMore: () => void;
    className?: string;
};

const VIEWPORT_OFFSET = 200;

export const InfiniteScrollTrigger = ({
    hasMore,
    loading,
    disabled,
    onLoadMore,
    className,
}: Props) => {
    const sentinelRef = useRef<HTMLDivElement | null>(null);
    const requestedRef = useRef(false);
    const wasNearEndRef = useRef(false);
    const frameRef = useRef<number | null>(null);

    const triggerLoadMore = useCallback(() => {
        if (!hasMore || disabled || requestedRef.current) {
            return;
        }

        requestedRef.current = true;
        onLoadMore();
    }, [disabled, hasMore, onLoadMore]);

    useEffect(() => {
        if (!disabled) {
            requestedRef.current = false;
        }
    }, [disabled]);

    useEffect(() => {
        if (!hasMore) {
            requestedRef.current = false;
        }
    }, [hasMore]);

    useEffect(() => {
        const isNearEnd = () => {
            const node = sentinelRef.current;
            if (!node) {
                return false;
            }

            const rect = node.getBoundingClientRect();
            return rect.top <= window.innerHeight + VIEWPORT_OFFSET;
        };

        const maybeLoadMore = () => {
            const nearEnd = isNearEnd();

            if (nearEnd && !wasNearEndRef.current) {
                triggerLoadMore();
            }

            wasNearEndRef.current = nearEnd;
        };

        const handleScroll = () => {
            if (window.scrollY <= 0) {
                wasNearEndRef.current = false;
                return;
            }

            if (frameRef.current !== null) {
                return;
            }

            frameRef.current = window.requestAnimationFrame(() => {
                frameRef.current = null;
                maybeLoadMore();
            });
        };

        window.addEventListener('scroll', handleScroll, { passive: true });

        return () => {
            if (frameRef.current !== null) {
                window.cancelAnimationFrame(frameRef.current);
            }
            window.removeEventListener('scroll', handleScroll);
        };
    }, [disabled, hasMore, triggerLoadMore]);

    if (!hasMore && !loading) {
        return null;
    }

    return (
        <div ref={sentinelRef} className={cn(s.loader, className, { [s.loading]: loading })}>
            {loading && (
                <>
                    <InlineLoader className={s.icon} />
                </>
            )}
        </div>
    );
};
