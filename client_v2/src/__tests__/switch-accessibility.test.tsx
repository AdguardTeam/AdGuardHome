import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Switch } from 'panel/common/controls/Switch';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';

describe('Switch accessibility', () => {
    it('exposes the given name to assistive technology', () => {
        render(() => (
            <Switch id="named" ariaLabel="Block ads" checked={false} onChange={() => {}} />
        ));

        expect(screen.getByRole('checkbox', { name: 'Block ads' })).toBeInTheDocument();
    });

    it('is reachable with the keyboard and toggles on Space', async () => {
        const onChange = vi.fn();
        render(() => (
            <Switch id="keyboard" ariaLabel="Block ads" checked={false} onChange={onChange} />
        ));

        const input = screen.getByRole('checkbox', { name: 'Block ads' });

        await userEvent.tab();
        expect(input).toHaveFocus();

        await userEvent.keyboard(' ');
        expect(onChange).toHaveBeenCalled();
    });

    it('does not take focus while disabled', async () => {
        render(() => (
            <Switch id="off" ariaLabel="Block ads" checked={false} disabled onChange={() => {}} />
        ));

        await userEvent.tab();
        expect(screen.getByRole('checkbox', { name: 'Block ads' })).not.toHaveFocus();
    });
});

describe('SwitchGroup accessibility', () => {
    it('names the control after its visible title', () => {
        render(() => (
            <SwitchGroup id="titled" title="Existing setting" checked={false} onChange={() => {}} />
        ));

        expect(screen.getByRole('checkbox', { name: 'Existing setting' })).toBeInTheDocument();
    });

    it('prefers an explicit name over the title', () => {
        render(() => (
            <SwitchGroup
                id="overridden"
                title="Visible title"
                ariaLabel="Spoken name"
                checked={false}
                onChange={() => {}}
            />
        ));

        expect(screen.getByRole('checkbox', { name: 'Spoken name' })).toBeInTheDocument();
    });
});
