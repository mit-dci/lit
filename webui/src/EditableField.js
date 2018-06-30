/**
 * Component that displays a string with an edit icon next to it. Clicking the icon makes the string editable
 */

import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Input from 'material-ui/Input';
import IconButton from 'material-ui/IconButton';
import EditIcon from '@material-ui/icons/Edit';
import CheckCircleIcon from '@material-ui/icons/CheckCircle';
import CancelIcon from '@material-ui/icons/Cancel';

const styles = theme => ({
});

class EditableField extends React.Component {
  state = {
    editing: false,
    newString: "",
  };

  handleEdit() {
    this.setState({editing: true});
  };

  handleCancel() {
    this.setState({editing: false});
  };

  handleConfirm() {
    this.props.handleSubmit(this.state.newString);
    this.setState(
      {
        editing: false,
        newString: "",
      });
  }

  handleChange(event) {
    this.setState({
        newString: event.target.value,
      });
  }

  render () {
    let output;

    if (this.state.editing) {
      output = (
        <span>
        <Input
          autoFocus
          id="string"
          type="text"
          onChange={this.handleChange.bind(this)}
        />
          <IconButton size="small" onClick={this.handleCancel.bind(this)}>
            <CancelIcon/>
          </IconButton>
          <IconButton size="small" onClick={this.handleConfirm.bind(this)}>
            <CheckCircleIcon/>
          </IconButton>
        </span>
          );
    } else {
      output = (
        <span>
          {this.props.string}
          <IconButton size="small" onClick={this.handleEdit.bind(this)}>
            <EditIcon/>
          </IconButton>
        </span>
      );
    }

    return output;
  }
}

EditableField.propTypes = {
  classes: PropTypes.object.isRequired,
  string: PropTypes.string.isRequired,
  handleSubmit: PropTypes.func.isRequired,
};

export default withStyles(styles)(EditableField);
