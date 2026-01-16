import React from 'react';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import styles from './styles.module.pcss';

type RequirementIconProps = {
    ok: boolean;
};

export const RequirementIcon = ({ ok }: RequirementIconProps) => {
    const iconName = ok ? 'check' : 'cross';
    const iconColor = ok ? 'green' : 'red';

    return <Icon icon={iconName} color={iconColor} />;
};

type PasswordRequirementsProps = {
    requirements: {
        minLength: boolean;
        allowedChars: boolean;
        lowercase: boolean;
        uppercase: boolean;
        match: boolean;
    };
    className?: string;
};

export const PasswordRequirements = ({ requirements, className }: PasswordRequirementsProps) => (
    <div className={className}>
        <h3 className={styles.bannerTitle}>{intl.getMessage('password_requirements')}</h3>

        <ul className={styles.bannerList}>
            <li className={styles.bannerItem}>
                <RequirementIcon ok={requirements.minLength} />
                {intl.getMessage('password_requirements_characters')}
            </li>

            <li className={styles.bannerItem}>
                <RequirementIcon ok={requirements.allowedChars} />
                {intl.getMessage('password_requirements_special')}
            </li>

            <li className={styles.bannerItem}>
                <RequirementIcon ok={requirements.lowercase} />
                {intl.getMessage('password_requirements_lowercase')}
            </li>

            <li className={styles.bannerItem}>
                <RequirementIcon ok={requirements.uppercase} />
                {intl.getMessage('password_requirements_uppercase')}
            </li>

            <li className={styles.bannerItem}>
                <RequirementIcon ok={requirements.match} />
                {intl.getMessage('password_requirements_match')}
            </li>
        </ul>
    </div>
);
