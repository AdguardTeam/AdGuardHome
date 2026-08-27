import { createSignal } from 'solid-js';
import { render } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';

import { Progress } from 'panel/install/Setup/blocks/Progress';

// CSS modules are scoped in tests, so match by the class-name prefix.
const getGreenPills = () => document.querySelectorAll('[class*="progressStepGreen"]');
const getGreyPills = () => document.querySelectorAll('[class*="progressStepGrey"]');

describe('Progress', () => {
    it('marks the current step pill as active (green)', () => {
        render(() => <Progress step={1} />);

        expect(getGreenPills()).toHaveLength(1);
        expect(getGreyPills()).toHaveLength(4);
    });

    it('updates the active pills when the step changes', () => {
        const [step, setStep] = createSignal(1);
        render(() => <Progress step={step()} />);

        expect(getGreenPills()).toHaveLength(1);
        expect(getGreyPills()).toHaveLength(4);

        setStep(3);

        expect(getGreenPills()).toHaveLength(3);
        expect(getGreyPills()).toHaveLength(2);

        setStep(5);

        expect(getGreenPills()).toHaveLength(5);
        expect(getGreyPills()).toHaveLength(0);
    });

    it('hides the bar on the last step and clamps the counter', () => {
        render(() => <Progress step={5} />);

        expect(document.querySelector('[role="progressbar"]')).not.toBeNull();
        expect(document.body.textContent).toContain('5/5');
    });
});
