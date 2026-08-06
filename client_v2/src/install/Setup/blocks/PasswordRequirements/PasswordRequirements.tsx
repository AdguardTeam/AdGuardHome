import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import styles from './styles.module.pcss';

type RequirementIconProps = {
    ok: boolean;
};

export const RequirementIcon = (props: RequirementIconProps) => {
    const iconName = () => (props.ok ? 'check' : 'cross');
    const iconColor = () => (props.ok ? 'green' : 'red');

    return <Icon icon={iconName()} color={iconColor()} />;
};

type PasswordRequirementsProps = {
    requirements: {
        minLength: boolean;
        allowedChars: boolean;
        lowercase: boolean;
        uppercase: boolean;
        match: boolean;
    };
    class?: string;
};

export const PasswordRequirements = (props: PasswordRequirementsProps) => (
    <div class={props.class}>
        <h3 class={styles.bannerTitle}>{intl.getMessage('password_requirements')}</h3>

        <ul class={styles.bannerList}>
            <li class={styles.bannerItem}>
                <RequirementIcon ok={props.requirements.minLength} />
                {intl.getMessage('password_requirements_characters')}
            </li>

            <li class={styles.bannerItem}>
                <RequirementIcon ok={props.requirements.allowedChars} />
                {intl.getMessage('password_requirements_special')}
            </li>

            <li class={styles.bannerItem}>
                <RequirementIcon ok={props.requirements.lowercase} />
                {intl.getMessage('password_requirements_lowercase')}
            </li>

            <li class={styles.bannerItem}>
                <RequirementIcon ok={props.requirements.uppercase} />
                {intl.getMessage('password_requirements_uppercase')}
            </li>

            <li class={styles.bannerItem}>
                <RequirementIcon ok={props.requirements.match} />
                {intl.getMessage('password_requirements_match')}
            </li>
        </ul>
    </div>
);
