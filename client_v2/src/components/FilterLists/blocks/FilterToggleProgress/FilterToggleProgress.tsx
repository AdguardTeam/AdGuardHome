import { Show } from 'solid-js';
import { Portal } from 'solid-js/web';
import { Dialog as ArkDialog } from '@ark-ui/solid';

import intl from 'panel/common/intl';
import { Loader } from 'panel/common/ui/Loader';
import { filteringState } from 'panel/stores/filtering';

import s from './FilterToggleProgress.module.pcss';

export const FilterToggleProgress = () => (
    <Show when={filteringState.processingConfigFilter}>
        <ArkDialog.Root
            open
            closeOnEscape={false}
            closeOnInteractOutside={false}
            modal
            preventScroll
            trapFocus
        >
            <Portal>
                <ArkDialog.Backdrop class={s.backdrop} />
                <ArkDialog.Positioner class={s.positioner}>
                    <ArkDialog.Content class={s.content} aria-busy="true">
                        <div aria-hidden="true">
                            <Loader class={s.loader} />
                        </div>
                        <ArkDialog.Title class={s.title}>
                            {intl.getMessage('filter_toggle_progress_title')}
                        </ArkDialog.Title>
                        <ArkDialog.Description class={s.description}>
                            {intl.getMessage('filter_toggle_progress_desc')}
                        </ArkDialog.Description>
                    </ArkDialog.Content>
                </ArkDialog.Positioner>
            </Portal>
        </ArkDialog.Root>
    </Show>
);
