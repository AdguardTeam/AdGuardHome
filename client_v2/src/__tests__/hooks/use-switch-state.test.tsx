import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@solidjs/testing-library';
import { createSignal, type Accessor } from 'solid-js';

import { useSwitchState } from 'panel/hooks/useSwitchState';

type HarnessProps = {
    value: Accessor<boolean>;
    processing: Accessor<boolean>;
    /** Stands in for a handler that rejects the toggle. */
    reject?: (setChecked: (next: boolean) => void) => void;
};

const Harness = (props: HarnessProps) => {
    const [checked, setChecked] = useSwitchState(
        () => props.value(),
        () => props.processing(),
    );

    return (
        <input
            id="switch"
            type="checkbox"
            checked={checked()}
            onChange={() => props.reject?.(setChecked)}
        />
    );
};

const getInput = () => document.querySelector('#switch') as HTMLInputElement;

describe('useSwitchState', () => {
    it('renders the store value', () => {
        render(() => <Harness value={() => true} processing={() => false} />);

        expect(getInput().checked).toBe(true);
    });

    it('reverts the DOM when a toggle is rejected', () => {
        render(() => (
            <Harness
                value={() => false}
                processing={() => false}
                reject={(setChecked) => setChecked(false)}
            />
        ));

        // The browser flips the checkbox itself, so the rejection has to put it
        // back even though the store value never changed.
        fireEvent.change(getInput(), { target: { checked: true } });

        expect(getInput().checked).toBe(false);
    });

    it('re-syncs with the store once the request finishes', () => {
        const [value, setValue] = createSignal(true);
        const [processing, setProcessing] = createSignal(false);

        render(() => <Harness value={value} processing={processing} />);
        expect(getInput().checked).toBe(true);

        // A save is in flight — the switch keeps showing the user's intent.
        setProcessing(true);
        setValue(false);
        expect(getInput().checked).toBe(true);

        // Once it finishes, the store wins.
        setProcessing(false);
        expect(getInput().checked).toBe(false);
    });
});
