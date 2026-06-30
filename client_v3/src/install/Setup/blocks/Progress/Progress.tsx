import { Show, For } from 'solid-js';
import intl from 'panel/common/intl';
import cn from 'clsx';
import { INSTALL_TOTAL_STEPS } from 'panel/helpers/constants';
import styles from './styles.module.pcss';

type Props = { step: number };

export const Progress = (props: Props) => {
    const totalProgressSteps = INSTALL_TOTAL_STEPS - 1;
    const progressStep = () => Math.min(props.step, totalProgressSteps);

    return (
        <Show when={props.step < INSTALL_TOTAL_STEPS}>
            <div class={styles.progress}>
                <div>
                    <div class={styles.message}>{intl.getMessage('install_step')}</div>
                    {progressStep()}/{totalProgressSteps}
                </div>

                <div
                    class={styles.progressWrap}
                    role="progressbar"
                    aria-valuenow={progressStep()}
                    aria-valuemin={1}
                    aria-valuemax={totalProgressSteps}
                >
                    <For each={Array.from({ length: totalProgressSteps }, (_, i) => i + 1)}>
                        {(installStep) => {
                            const isDoneOrCurrent = installStep <= progressStep();
                            return (
                                <div
                                    class={cn(styles.progressStep, {
                                        [styles.progressStepGreen]: isDoneOrCurrent,
                                        [styles.progressStepGrey]: !isDoneOrCurrent,
                                    })}
                                />
                            );
                        }}
                    </For>
                </div>
            </div>
        </Show>
    );
};
