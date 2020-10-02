import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactModal from 'react-modal';
import { withTranslation } from 'react-i18next';

import { MODAL_TYPE } from '../../helpers/constants';
import Form from './Form';
import '../ui/Modal.css';
import { getMap } from '../../helpers/helpers';

ReactModal.setAppElement('#root');

const MODAL_TYPE_TO_TITLE_TYPE_MAP = {
    [MODAL_TYPE.EDIT_FILTERS]: 'edit',
    [MODAL_TYPE.ADD_FILTERS]: 'new',
    [MODAL_TYPE.SELECT_MODAL_TYPE]: 'new',
    [MODAL_TYPE.CHOOSE_FILTERING_LIST]: 'choose',
};

/**
 * @param modalType {'EDIT_FILTERS' | 'ADD_FILTERS' | 'CHOOSE_FILTERING_LIST'}
 * @param whitelist {boolean}
 * @returns {'new_allowlist' | 'edit_allowlist' | 'choose_allowlist' |
 *           'new_blocklist' | 'edit_blocklist' | 'choose_blocklist' | null}
 */
const getTitle = (modalType, whitelist) => {
    const titleType = MODAL_TYPE_TO_TITLE_TYPE_MAP[modalType];
    if (!titleType) {
        return null;
    }
    return `${titleType}_${whitelist ? 'allowlist' : 'blocklist'}`;
};

const getSelectedValues = (filters, catalogSourcesToIdMap) => filters.reduce((acc, { url }) => {
    if (Object.prototype.hasOwnProperty.call(catalogSourcesToIdMap, url)) {
        const fieldId = `filter${catalogSourcesToIdMap[url]}`;
        acc.selectedFilterIds[fieldId] = true;
        acc.selectedSources[url] = true;
    }
    return acc;
}, {
    selectedFilterIds: {},
    selectedSources: {},
});

class Modal extends Component {
    closeModal = () => {
        this.props.toggleFilteringModal();
    };

    render() {
        const {
            isOpen,
            processingAddFilter,
            processingConfigFilter,
            handleSubmit,
            modalType,
            currentFilterData,
            whitelist,
            toggleFilteringModal,
            filters,
            t,
            filtersCatalog,
        } = this.props;

        let initialValues;
        let selectedSources;
        switch (modalType) {
            case MODAL_TYPE.EDIT_FILTERS:
                initialValues = currentFilterData;
                break;
            case MODAL_TYPE.CHOOSE_FILTERING_LIST: {
                const catalogSourcesToIdMap = getMap(Object.values(filtersCatalog.filters), 'source', 'id');

                const selectedValues = getSelectedValues(filters, catalogSourcesToIdMap);
                initialValues = selectedValues.selectedFilterIds;
                selectedSources = selectedValues.selectedSources;
                break;
            }
            default:
                break;
        }

        const title = t(getTitle(modalType, whitelist));

        return (
            <ReactModal
                className="Modal__Bootstrap modal-dialog modal-dialog-centered"
                closeTimeoutMS={0}
                isOpen={isOpen}
                onRequestClose={this.closeModal}
            >
                <div className="modal-content">
                    <div className="modal-header">
                        {title && <h4 className="modal-title">{title}</h4>}
                        <button type="button" className="close" onClick={this.closeModal}>
                            <span className="sr-only">Close</span>
                        </button>
                    </div>
                    <Form
                        selectedSources={selectedSources}
                        initialValues={initialValues}
                        modalType={modalType}
                        onSubmit={handleSubmit}
                        processingAddFilter={processingAddFilter}
                        processingConfigFilter={processingConfigFilter}
                        closeModal={this.closeModal}
                        whitelist={whitelist}
                        toggleFilteringModal={toggleFilteringModal}
                    />
                </div>
            </ReactModal>
        );
    }
}

Modal.propTypes = {
    toggleFilteringModal: PropTypes.func.isRequired,
    isOpen: PropTypes.bool.isRequired,
    addFilter: PropTypes.func.isRequired,
    isFilterAdded: PropTypes.bool.isRequired,
    processingAddFilter: PropTypes.bool.isRequired,
    processingConfigFilter: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    modalType: PropTypes.string.isRequired,
    currentFilterData: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
    whitelist: PropTypes.bool,
    filters: PropTypes.array.isRequired,
    filtersCatalog: PropTypes.object,
};

export default withTranslation()(Modal);
