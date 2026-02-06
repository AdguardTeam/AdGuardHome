import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import RulesTable from './RulesTable';
import RuleModal from './RuleModal';
import AutoCreateTable from './AutoCreateTable';
import AutoCreateModal, { IpsetDefinition } from './AutoCreateModal';
import { parseIPSetRule, isDuplicateRule, validateIPSetRule } from '../../../helpers/ipset';
import { Radio } from '../../ui/Controls/Radio';
import { Input } from '../../ui/Controls/Input';
import { Checkbox } from '../../ui/Controls/Checkbox';

interface IpsetCreateConfig {
    enabled: boolean;
    sets: IpsetDefinition[];
}

interface FormProps {
    initialRules: string[];
    initialFilePath: string;
    initialIpsetCreate: IpsetCreateConfig | null;
    onSubmit: (data: { ipset: string[]; ipset_file: string; ipset_create: IpsetCreateConfig | null }) => void;
    processing: boolean;
}

type StorageMode = 'config' | 'file';

const Form: React.FC<FormProps> = ({ initialRules, initialFilePath, initialIpsetCreate, onSubmit, processing }) => {
    const { t } = useTranslation();

    // Determine initial mode
    const initialMode: StorageMode = initialFilePath && initialFilePath.trim() !== '' ? 'file' : 'config';

    const [mode, setMode] = useState<StorageMode>(initialMode);
    const [rules, setRules] = useState<string[]>(initialRules);
    const [filePath, setFilePath] = useState<string>(initialFilePath);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [editingIndex, setEditingIndex] = useState<number | null>(null);
    const [isDirty, setIsDirty] = useState(false);

    // AutoCreate state
    const [autoCreateEnabled, setAutoCreateEnabled] = useState(initialIpsetCreate?.enabled || false);
    const [autoCreateSets, setAutoCreateSets] = useState<IpsetDefinition[]>(initialIpsetCreate?.sets || []);
    const [isAutoCreateModalOpen, setIsAutoCreateModalOpen] = useState(false);
    const [editingAutoCreateIndex, setEditingAutoCreateIndex] = useState<number | null>(null);

    // Update when initial values change
    useEffect(() => {
        setRules(initialRules);
        setFilePath(initialFilePath);
        setAutoCreateEnabled(initialIpsetCreate?.enabled || false);
        setAutoCreateSets(initialIpsetCreate?.sets || []);
        const newMode = initialFilePath && initialFilePath.trim() !== '' ? 'file' : 'config';
        setMode(newMode);
        setIsDirty(false);
    }, [initialRules, initialFilePath, initialIpsetCreate]);

    const handleModeChange = (newMode: StorageMode) => {
        setMode(newMode);
        setIsDirty(true);
    };

    const handleAddRule = () => {
        setEditingIndex(null);
        setIsModalOpen(true);
    };

    const handleEditRule = (index: number, _rule: string) => {
        setEditingIndex(index);
        setIsModalOpen(true);
    };

    const handleSaveRule = (newRule: string) => {
        // Validate rule
        const error = validateIPSetRule(newRule);
        if (error) {
            alert(`Invalid rule: ${error}`);
            return;
        }

        // Check for duplicates (exclude current rule if editing)
        const otherRules = editingIndex !== null
            ? rules.filter((_, i) => i !== editingIndex)
            : rules;

        if (isDuplicateRule(newRule, otherRules)) {
            alert(t('ipset_duplicate_rule'));
            return;
        }

        if (editingIndex !== null) {
            // Edit existing rule
            const newRules = [...rules];
            newRules[editingIndex] = newRule;
            setRules(newRules);
        } else {
            // Add new rule
            setRules([...rules, newRule]);
        }

        setIsDirty(true);
    };

    const handleDeleteRule = (index: number) => {
        if (window.confirm(t('ipset_confirm_delete'))) {
            const newRules = rules.filter((_, i) => i !== index);
            setRules(newRules);
            setIsDirty(true);
        }
    };

    const handleFilePathChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setFilePath(e.target.value);
        setIsDirty(true);
    };

    const handleAutoCreateEnabledChange = () => {
        setAutoCreateEnabled(!autoCreateEnabled);
        setIsDirty(true);
    };

    const handleAddAutoCreateSet = () => {
        setEditingAutoCreateIndex(null);
        setIsAutoCreateModalOpen(true);
    };

    const handleEditAutoCreateSet = (index: number, _definition: IpsetDefinition) => {
        setEditingAutoCreateIndex(index);
        setIsAutoCreateModalOpen(true);
    };

    const handleSaveAutoCreateSet = (definitions: IpsetDefinition[]) => {
        if (editingAutoCreateIndex !== null) {
            // When editing, replace the single item
            const newSets = [...autoCreateSets];
            newSets[editingAutoCreateIndex] = definitions[0];
            setAutoCreateSets(newSets);
        } else {
            // When adding, append all new definitions
            setAutoCreateSets([...autoCreateSets, ...definitions]);
        }
        setIsDirty(true);
    };

    const handleDeleteAutoCreateSet = (index: number) => {
        if (window.confirm(t('ipset_autocreate_confirm_delete'))) {
            const newSets = autoCreateSets.filter((_, i) => i !== index);
            setAutoCreateSets(newSets);
            setIsDirty(true);
        }
    };

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();

        const ipsetCreate: IpsetCreateConfig = {
            enabled: autoCreateEnabled,
            sets: autoCreateSets,
        };

        if (mode === 'file') {
            if (!filePath || filePath.trim() === '') {
                alert(t('ipset_file_path_required'));
                return;
            }
            onSubmit({ ipset: [], ipset_file: filePath.trim(), ipset_create: ipsetCreate });
        } else {
            // Validate all rules
            const invalidRule = rules.find((rule) => validateIPSetRule(rule) !== undefined);
            if (invalidRule) {
                const error = validateIPSetRule(invalidRule);
                alert(`Invalid rule "${invalidRule}": ${error}`);
                return;
            }
            onSubmit({ ipset: rules, ipset_file: '', ipset_create: ipsetCreate });
        }

        setIsDirty(false);
    };

    const modeOptions = [
        { value: 'config', label: t('ipset_mode_config') },
        { value: 'file', label: t('ipset_mode_file') },
    ];

    const editingRule = editingIndex !== null ? rules[editingIndex] : null;
    const parsedEditingRule = editingRule ? parseIPSetRule(editingRule) : null;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <label className="form__label form__label--with-desc">
                            {t('ipset_storage_mode')}
                        </label>
                        <div className="form__desc form__desc--top">{t('ipset_storage_mode_desc')}</div>
                        <div className="custom-controls-stacked">
                            <Radio
                                name="storage_mode"
                                value={mode}
                                options={modeOptions}
                                disabled={processing}
                                onChange={(value) => handleModeChange(value as StorageMode)}
                            />
                        </div>
                    </div>
                </div>

                {mode === 'file' ? (
                    <div className="col-12 col-md-7">
                        <div className="form__group form__group--settings">
                            <Input
                                name="ipset_file"
                                value={filePath}
                                onChange={handleFilePathChange}
                                label={t('ipset_file_path')}
                                desc={t('ipset_file_path_desc')}
                                placeholder="/etc/adguardhome/ipset.conf"
                                disabled={processing}
                            />
                        </div>
                    </div>
                ) : (
                    <div className="col-12">
                        <div className="form__group form__group--settings">
                            <label className="form__label">{t('ipset_rules')}</label>
                            <div className="form__desc mb-3">{t('ipset_rules_desc')}</div>

                            <button
                                type="button"
                                className="btn btn-success btn-sm mb-3"
                                onClick={handleAddRule}
                                disabled={processing}>
                                + {t('ipset_add_rule')}
                            </button>

                            <RulesTable
                                rules={rules}
                                onEdit={handleEditRule}
                                onDelete={handleDeleteRule}
                                disabled={processing}
                            />
                        </div>
                    </div>
                )}

                <div className="col-12">
                    <hr className="my-4" />
                    <div className="form__group form__group--settings">
                        <label className="form__label form__label--with-desc">
                            {t('ipset_autocreate_title')}
                        </label>
                        <div className="form__desc form__desc--top mb-3">
                            {t('ipset_autocreate_desc')}
                        </div>
                        <Checkbox
                            name="autocreate_enabled"
                            value={autoCreateEnabled}
                            title={t('ipset_autocreate_enable')}
                            disabled={processing}
                            onChange={handleAutoCreateEnabledChange}
                        />
                    </div>

                    {autoCreateEnabled && (
                        <div className="form__group form__group--settings mt-4">
                            <label className="form__label">{t('ipset_autocreate_sets')}</label>
                            <div className="form__desc mb-3">{t('ipset_autocreate_sets_desc')}</div>

                            <button
                                type="button"
                                className="btn btn-success btn-sm mb-3"
                                onClick={handleAddAutoCreateSet}
                                disabled={processing}>
                                + {t('ipset_autocreate_add')}
                            </button>

                            <AutoCreateTable
                                definitions={autoCreateSets}
                                onEdit={handleEditAutoCreateSet}
                                onDelete={handleDeleteAutoCreateSet}
                                disabled={processing}
                            />
                        </div>
                    )}
                </div>

                <div className="col-12">
                    <div className="alert alert-info">
                        <strong>{t('ipset_info_title')}:</strong>
                        <p className="mb-1">{t('ipset_info_desc')}</p>
                        <ul className="mb-0">
                            <li>{t('ipset_info_linux_only')}</li>
                            <li>{t('ipset_info_format')}</li>
                        </ul>
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standard btn-large"
                disabled={!isDirty || processing}>
                {t('save_btn')}
            </button>

            <AutoCreateModal
                isOpen={isAutoCreateModalOpen}
                onClose={() => setIsAutoCreateModalOpen(false)}
                onSave={handleSaveAutoCreateSet}
                initialDefinition={editingAutoCreateIndex !== null ? autoCreateSets[editingAutoCreateIndex] : null}
                title={editingAutoCreateIndex !== null ? t('ipset_autocreate_edit') : t('ipset_autocreate_add')}
            />

            <RuleModal
                isOpen={isModalOpen}
                onClose={() => setIsModalOpen(false)}
                onSave={handleSaveRule}
                initialDomains={parsedEditingRule?.domains.join(',') || ''}
                initialIPSets={parsedEditingRule?.ipsets.join(',') || ''}
                title={editingIndex !== null ? t('ipset_edit_rule') : t('ipset_add_rule')}
            />
        </form>
    );
};

export default Form;
