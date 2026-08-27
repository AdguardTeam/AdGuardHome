import { For, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';

import s from './styles.module.pcss';

const STEPS = [1, 2, 3] as const;

type Props = {
    step: 1 | 2 | 3;
    onGoBack: () => void;
};

/**
 * Wizard header — "Go back" link (steps 2-3) and the 3-segment progress
 * bar per the TLS setup wizard design.  The design's switch/arrow icons
 * are hidden artifacts and intentionally omitted.
 */
export const WizardSteps = (props: Props) => (
    <div
        class={s.header}
        role="progressbar"
        aria-valuemin={1}
        aria-valuemax={3}
        aria-valuenow={props.step}
    >
        <Show when={props.step > 1}>
            <button type="button" class={s.goBack} onClick={() => props.onGoBack()}>
                {intl.getMessage('go_back')}
            </button>
        </Show>
        <div class={s.steps}>
            <For each={STEPS}>
                {(step) => (
                    <div
                        class={cn(s.pill, {
                            [s.pill_active]: step <= props.step,
                        })}
                    />
                )}
            </For>
        </div>
    </div>
);
