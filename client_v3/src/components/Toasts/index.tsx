import { For } from 'solid-js';
import { TransitionGroup } from 'solid-transition-group';

import { TOAST_TRANSITION_TIMEOUT } from '../../helpers/constants';

import Toast from './Toast';
import './Toast.pcss';
import { toastsState } from 'panel/stores/toasts';

const Toasts = () => {
    return (
        <div class="toasts">
            <For each={toastsState.notices}>{(toast: any) => <Toast {...toast} />}</For>
        </div>
    );
};

export default Toasts;
