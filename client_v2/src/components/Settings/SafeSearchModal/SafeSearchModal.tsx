import { createEffect, For, on } from 'solid-js';
import { createStore } from 'solid-js/store';

import { Checkbox } from 'panel/common/controls/Checkbox';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { SAFE_SEARCH_PROVIDERS } from 'panel/helpers/constants';
import { getSafeSearchProviderTitle } from '../helpers';
import intl from 'panel/common/intl';

import s from './SafeSearchModal.module.pcss';

type Props = {
    open: boolean;
    onClose: () => void;
    providers: Record<string, boolean>;
    enabled: boolean;
    processing: boolean;
    onSave: (providers: Record<string, boolean>) => void;
};

export const SafeSearchModal = (props: Props) => {
    const [selected, setSelected] = createStore<Record<string, boolean>>({});

    createEffect(
        on(
            () => props.open,
            (open) => {
                if (!open) return;
                const entries = Object.keys(SAFE_SEARCH_PROVIDERS).map((key) => [
                    key,
                    props.providers[key] ?? false,
                ]);
                setSelected(Object.fromEntries(entries));
            },
        ),
    );

    const handleSave = () => {
        const result: Record<string, boolean> = {};
        Object.keys(SAFE_SEARCH_PROVIDERS).forEach((key) => {
            result[key] = selected[key] ?? false;
        });
        props.onSave(result);
    };

    return (
        <ConfigDialog
            open={props.open}
            title={intl.getMessage('settings_safe_search')}
            onClose={props.onClose}
            onSubmit={handleSave}
            processing={props.processing}
            description={intl.getMessage('settings_safe_search_desc')}
        >
            <div class={s.providersGrid}>
                <For each={Object.keys(SAFE_SEARCH_PROVIDERS)}>
                    {(key) => (
                        <Checkbox
                            id={`safesearch-${key}`}
                            checked={selected[key] ?? false}
                            disabled={props.processing}
                            onChange={() => setSelected(key, !selected[key])}
                        >
                            {getSafeSearchProviderTitle(key)}
                        </Checkbox>
                    )}
                </For>
            </div>
        </ConfigDialog>
    );
};
