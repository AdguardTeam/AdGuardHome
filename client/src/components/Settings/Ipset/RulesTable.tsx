import React from 'react';
import { useTranslation } from 'react-i18next';

import { parseIPSetRule } from '../../../helpers/ipset';

interface RulesTableProps {
    rules: string[];
    onEdit: (index: number, rule: string) => void;
    onDelete: (index: number) => void;
    disabled?: boolean;
}

const RulesTable: React.FC<RulesTableProps> = ({ rules, onEdit, onDelete, disabled = false }) => {
    const { t } = useTranslation();

    if (rules.length === 0) {
        return (
            <div className="alert alert-info">
                {disabled ? t('ipset_rules_from_file') : t('ipset_no_rules')}
            </div>
        );
    }

    return (
        <div className="table-responsive">
            <table className="table table-bordered">
                <thead>
                    <tr>
                        <th>{t('ipset_domains')}</th>
                        <th>{t('ipset_names')}</th>
                        <th style={{ width: '120px' }}>{t('actions_table_header')}</th>
                    </tr>
                </thead>
                <tbody>
                    {rules.map((rule, index) => {
                        const parsed = parseIPSetRule(rule);
                        if (!parsed) {
                            return (
                                <tr key={index}>
                                    <td colSpan={3} className="text-danger">
                                        {t('ipset_invalid_rule')}: {rule}
                                    </td>
                                </tr>
                            );
                        }

                        return (
                            <tr key={index}>
                                <td>
                                    <code>{parsed.domains.join(', ')}</code>
                                </td>
                                <td>
                                    <code>{parsed.ipsets.join(', ')}</code>
                                </td>
                                <td>
                                    <button
                                        type="button"
                                        className="btn btn-icon btn-sm btn-outline-primary mr-2"
                                        onClick={() => onEdit(index, rule)}
                                        disabled={disabled}
                                        title={t('edit')}>
                                        <svg className="icons icon--small">
                                            <use xlinkHref="#edit" />
                                        </svg>
                                    </button>
                                    <button
                                        type="button"
                                        className="btn btn-icon btn-sm btn-outline-danger"
                                        onClick={() => onDelete(index)}
                                        disabled={disabled}
                                        title={t('delete')}>
                                        <svg className="icons icon--small">
                                            <use xlinkHref="#delete" />
                                        </svg>
                                    </button>
                                </td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
};

export default RulesTable;
