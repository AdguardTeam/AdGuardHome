import { createEffect, onCleanup, Show, untrack } from 'solid-js';
import cn from 'clsx';

import { InlineLoader } from 'panel/common/ui/Loader';

import s from './InfiniteScrollTrigger.module.pcss';

type Props = {
    hasMore: boolean;
    loading: boolean;
    disabled: boolean;
    onLoadMore: () => void;
    resetToken?: string;
    class?: string;
};

const VIEWPORT_OFFSET = 200;

export const InfiniteScrollTrigger = (props: Props) => {
    let sentinelEl: HTMLDivElement | undefined;
    let requestedRef = false;
    let wasNearEndRef = false;
    let frameRef: number | null = null;

    const triggerLoadMore = () => {
        if (!untrack(() => props.hasMore) || untrack(() => props.disabled) || requestedRef) {
            return;
        }
        requestedRef = true;
        untrack(() => props).onLoadMore();
    };

    createEffect(() => {
        if (!props.disabled) {
            requestedRef = false;
            wasNearEndRef = false;
        }
    });

    createEffect(() => {
        if (!props.hasMore) {
            requestedRef = false;
        }
    });

    createEffect(() => {
        // Track resetToken to trigger re-read
        void props.resetToken;
        requestedRef = false;
        wasNearEndRef = false;
    });

    createEffect(() => {
        // Track dependencies
        void props.disabled;
        void props.hasMore;
        void props.resetToken;

        const isNearEnd = () => {
            if (!sentinelEl) {
                return false;
            }

            const rect = sentinelEl.getBoundingClientRect();

            // Ignore hidden elements (e.g. inside display:none containers)
            if (rect.width === 0 && rect.height === 0) {
                return false;
            }

            return rect.top <= window.innerHeight + VIEWPORT_OFFSET;
        };

        const maybeLoadMore = () => {
            const nearEnd = isNearEnd();

            if (nearEnd && !wasNearEndRef) {
                triggerLoadMore();
            }

            wasNearEndRef = nearEnd;
        };

        const scheduleMaybeLoadMore = () => {
            if (frameRef !== null) {
                return;
            }

            frameRef = window.requestAnimationFrame(() => {
                frameRef = null;
                maybeLoadMore();
            });
        };

        const handleScroll = () => {
            if (window.scrollY <= 0) {
                wasNearEndRef = false;
            }
            scheduleMaybeLoadMore();
        };

        const handleResize = () => {
            scheduleMaybeLoadMore();
        };

        scheduleMaybeLoadMore();

        window.addEventListener('scroll', handleScroll, { passive: true });
        window.addEventListener('resize', handleResize);

        onCleanup(() => {
            if (frameRef !== null) {
                window.cancelAnimationFrame(frameRef);
                frameRef = null;
            }
            window.removeEventListener('scroll', handleScroll);
            window.removeEventListener('resize', handleResize);
        });
    });

    return (
        <Show when={props.hasMore || props.loading}>
            <div
                ref={sentinelEl}
                data-testid="query-log-infinite-scroll-trigger"
                class={cn(s.loader, props.class, { [s.loading]: props.loading })}
            >
                <Show when={props.loading}>
                    <InlineLoader class={s.icon} />
                </Show>
            </div>
        </Show>
    );
};
