import { type JSX, Show } from 'solid-js';
import { modalsState } from 'panel/stores/modals';

interface Props {
    id: string;
    children: JSX.Element;
}

export const ModalWrapper = (props: Props) => {
    return <Show when={modalsState.modalId === props.id}>{props.children}</Show>;
};
