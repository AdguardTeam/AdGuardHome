import { connect } from 'react-redux';
import { getRewritesList, addRewrite, deleteRewrite, updateRewrite, toggleRewritesModal, updateRewriteSettings, getRewriteSettings } from '../actions/rewrites';

import Rewrites from '../components/Filters/Rewrites';
import { RootState } from '../initialState';

const mapStateToProps = (state: RootState) => {
    const { rewrites } = state;
    const props = { rewrites };
    return props;
};

type DispatchProps = {
    getRewritesList: () => (dispatch: any) => void;
    toggleRewritesModal: (...args: unknown[]) => unknown;
    addRewrite: (...args: unknown[]) => unknown;
    deleteRewrite: (...args: unknown[]) => unknown;
    updateRewrite: (...args: unknown[]) => unknown;
    updateRewriteSettings: (...args: unknown[]) => unknown;
    getRewriteSettings: () => (dispatch: any) => void;
}

const mapDispatchToProps: DispatchProps = {
    getRewritesList,
    addRewrite,
    deleteRewrite,
    updateRewrite,
    toggleRewritesModal,
    updateRewriteSettings,
    getRewriteSettings,
};

export default connect(mapStateToProps, mapDispatchToProps)(Rewrites);
