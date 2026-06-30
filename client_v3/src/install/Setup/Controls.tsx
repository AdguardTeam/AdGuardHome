import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import { installState, nextStep, prevStep } from 'panel/stores/install';
import styles from './styles.module.pcss';

type Props = {
    invalid?: boolean;
    ip?: string;
    port?: number;
    isDirty?: boolean;
    isValid?: boolean;
    openDashboard?: (ip: string, port: number) => void;
};

export const Controls = (props: Props) => {
    const handleNextStep = () => {
        nextStep();
    };

    const handlePrevStep = () => {
        prevStep();
    };

    const renderPrevButton = (step: number) => {
        switch (step) {
            case 2:
            case 3:
            case 4:
            case 5:
                return (
                    <Button
                        id="install_back"
                        type="button"
                        size="small"
                        variant="secondary"
                        class={styles.button}
                        onClick={handlePrevStep}
                    >
                        {intl.getMessage('back')}
                    </Button>
                );
            case 6:
                return false;
            default:
                return false;
        }
    };

    const renderNextButton = (step: number) => {
        const isNextDisabled = props.invalid === true || props.isValid === false;

        switch (step) {
            case 1:
                return (
                    <Button
                        id="install_get_started"
                        type="button"
                        onClick={handleNextStep}
                        size="small"
                        variant="primary"
                        class={styles.button}
                    >
                        {intl.getMessage('setup_guide_greeting_button')}
                    </Button>
                );
            case 2:
            case 3:
            case 4:
                return (
                    <Button
                        id="install_next"
                        type="submit"
                        size="small"
                        variant="primary"
                        class={styles.button}
                        disabled={isNextDisabled}
                    >
                        {intl.getMessage('next')}
                    </Button>
                );
            case 5:
                return (
                    <Button
                        id="install_next"
                        type="button"
                        onClick={handleNextStep}
                        size="small"
                        variant="primary"
                        class={styles.button}
                        disabled={isNextDisabled}
                    >
                        {intl.getMessage('next')}
                    </Button>
                );
            case 6:
                return (
                    <Button
                        id="open_dashboard"
                        type="button"
                        size="small"
                        variant="primary"
                        class={styles.button}
                        onClick={() => {
                            if (props.openDashboard && props.ip && props.port) {
                                props.openDashboard(props.ip, props.port);
                            }
                        }}
                    >
                        {intl.getMessage('open_dashboard')}
                    </Button>
                );
            default:
                return false;
        }
    };

    return (
        <div class={styles.nav}>
            {renderNextButton(installState.step)}
            {renderPrevButton(installState.step)}
        </div>
    );
};
