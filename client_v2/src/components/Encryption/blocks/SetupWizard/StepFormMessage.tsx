import { Show } from 'solid-js';

import { InlineMessage } from './InlineMessage';
import type { StepMessage } from './mapStepResult';

type Props = {
    message?: StepMessage;
    class?: string;
};

/**
 * Form-level step message — for messages that cannot be attached to a field
 * visible on the current step, e.g. a failed save.
 */
export const StepFormMessage = (props: Props) => (
    <Show when={props.message}>
        <InlineMessage
            kind={props.message?.kind ?? 'error'}
            data-testid="tls-setup-form-message"
        >
            {props.message?.message}
        </InlineMessage>
    </Show>
);
