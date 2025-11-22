import React from 'react';
import { useTranslation } from 'react-i18next';

import type { IpsetDefinition } from './AutoCreateModal';

interface AutoCreateTableProps {
    definitions: IpsetDefinition[];
    onEdit: (index: number, definition: IpsetDefinition) => void;
    onDelete: (index: number) => void;
    disabled?: boolean;
}

const AutoCreateTable: React.FC<AutoCreateTableProps> = ({
    definitions,
    onEdit,
    onDelete,
    disabled = false,
}) => {
    const { t } = useTranslation();

    if (definitions.length === 0) {
        return (
            <div className="alert alert-info">
                {t('ipset_autocreate_no_sets')}
            </div>
        );
    }

    const getTypeLabel = (type: string) => {
        switch (type) {
            case 'hash:ip':
                return t('ipset_autocreate_type_ip');
            case 'hash:net':
                return t('ipset_autocreate_type_net');
            default:
                return type;
        }
    };

    const getFamilyLabel = (family: string) => {
        switch (family) {
            case 'inet':
                return t('ipset_autocreate_family_ipv4');
            case 'inet6':
                return t('ipset_autocreate_family_ipv6');
            default:
                return family;
        }
    };

    return (
        <table className="table table-bordered">
            <thead>
                <tr>
                    <th>{t('ipset_autocreate_name')}</th>
                    <th>{t('ipset_autocreate_type')}</th>
                    <th>{t('ipset_autocreate_family')}</th>
                    <th>{t('ipset_autocreate_timeout')}</th>
                    <th style={{ width: '120px' }}>{t('actions_table_header')}</th>
                </tr>
            </thead>
            <tbody>
                {definitions.map((def, index) => (
                    <tr key={index}>
                        <td>{def.name}</td>
                        <td>{getTypeLabel(def.type)}</td>
                        <td>{getFamilyLabel(def.family)}</td>
                        <td>{def.timeout === 0 ? t('disabled') : `${def.timeout}s`}</td>
                        <td>
                            <button
                                type="button"
                                className="btn btn-icon btn-sm btn-outline-primary mr-2"
                                onClick={() => onEdit(index, def)}
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
                ))}
            </tbody>
        </table>
    );
};

export default AutoCreateTable;
