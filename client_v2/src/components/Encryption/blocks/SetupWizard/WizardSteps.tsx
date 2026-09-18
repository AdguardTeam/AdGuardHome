import { For, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import s from './styles.module.pcss';

const STEPS = [1, 2, 3] as const;

type Props = {
    step: 1 | 2 | 3;
    onGoBack: () => void;
};

export const WizardSteps = (props: Props) => (
    <div
        class={s.header}
        role="progressbar"
        aria-valuemin={1}
        aria-valuemax={3}
        aria-valuenow={props.step}
        data-testid="tls-setup-progress"
    >
        <Show when={props.step > 1}>
            <button
                type="button"
                class={cn(theme.text.t3, s.goBack)}
                onClick={() => props.onGoBack()}
                data-testid="tls-setup-back"
            >
                {intl.getMessage('go_back')}
            </button>
        </Show>
        <div class={s.steps} data-testid="tls-setup-progress-pills">
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
