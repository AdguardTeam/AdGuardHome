import React, { Component, Fragment } from 'react';
import { Trans, withTranslation } from 'react-i18next';
import cn from 'classnames';

import Table from './Table';

import Modal from './Modal';

import Card from '../../ui/Card';

import PageTitle from '../../ui/PageTitle';
import { MODAL_TYPE } from '../../../helpers/constants';
import { RewritesData } from '../../../initialState';

interface RewritesProps {
    t: (...args: unknown[]) => string;
    getRewritesList: () => (dispatch: any) => void;
    toggleRewritesModal: (...args: unknown[]) => unknown;
    addRewrite: (...args: unknown[]) => unknown;
    deleteRewrite: (...args: unknown[]) => unknown;
    updateRewrite: (...args: unknown[]) => unknown;
    updateRewriteSettings: (...args: unknown[]) => unknown;
    getRewriteSettings: () => (dispatch: any) => void;
    rewrites: RewritesData;
}

class Rewrites extends Component<RewritesProps> {
    componentDidMount() {
        this.props.getRewritesList();
        this.props.getRewriteSettings();
    }

    handleDelete = (values: any) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('rewrite_confirm_delete', { key: values.domain }))) {
            this.props.deleteRewrite(values);
        }
    };

    handleSubmit = (values: any) => {
        const { modalType, currentRewrite } = this.props.rewrites;

        if (modalType === MODAL_TYPE.EDIT_REWRITE && currentRewrite) {
            this.props.updateRewrite({
                target: currentRewrite,
                update: values,
            });
        } else {
            this.props.addRewrite(values);
        }
    };

    toggleRewrite = (currentRewrite: any) => {
        const updatedRewrite = { ...currentRewrite, enabled: !currentRewrite.enabled };

        this.props.updateRewrite({
            target: currentRewrite,
            update: updatedRewrite,
        });
    };

    toggleRewriteSettings = () => {
        const { enabled } = this.props.rewrites.settings;

        this.props.updateRewriteSettings({ enabled: !enabled });
    };

    render() {
        const {
            t,
            rewrites,
            toggleRewritesModal,
        } = this.props;

        const {
            list,
            isModalOpen,
            processing,
            processingAdd,
            processingDelete,
            processingUpdate,
            modalType,
            currentRewrite,
            settings
        } = rewrites;

        const isEnabledSettings = settings.enabled;

        return (
            <Fragment>
                <PageTitle title={t('dns_rewrites')} subtitle={t('rewrite_desc')} />

                <div className={cn(isEnabledSettings ? 'text-success' : 'text-warning', 'mb-2')}>
                    {isEnabledSettings ? t('rewrites_enabled_table_header') : t('rewrites_disabled_table_header')}
                </div>

                <Card id="rewrites" bodyType="card-body box-body--settings">
                    <Fragment>
                        <Table
                            list={list}
                            processing={processing}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                            processingUpdate={processingUpdate}
                            handleDelete={this.handleDelete}
                            toggleRewritesModal={toggleRewritesModal}
                            toggleRewrite={this.toggleRewrite}
                            settings={settings}
                        />

                        <div className="card-actions">
                            <button
                                data-testid="add-rewrite"
                                type="button"
                                className="btn btn-success btn-standard  mr-2"
                                onClick={() => toggleRewritesModal({ type: MODAL_TYPE.ADD_REWRITE })}
                                disabled={processingAdd}>
                                <Trans>rewrite_add</Trans>
                            </button>

                            <button
                                data-testid="toggle-rewrite-settings"
                                type="button"
                                className="btn btn-primary btn-standard"
                                onClick={() => this.toggleRewriteSettings()}
                                disabled={processingUpdate}>
                                <Trans>{isEnabledSettings ? 'disable_rewrites' : 'enable_rewrites'}</Trans>
                            </button>
                        </div>

                        <Modal
                            isModalOpen={isModalOpen}
                            modalType={modalType}
                            toggleRewritesModal={toggleRewritesModal}
                            handleSubmit={this.handleSubmit}
                            processingAdd={processingAdd}
                            processingDelete={processingDelete}
                            currentRewrite={currentRewrite}
                        />
                    </Fragment>
                </Card>
            </Fragment>
        );
    }
}

export default withTranslation()(Rewrites);
