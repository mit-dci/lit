/**
 * Template for a new Material-UI component
 */

import React from 'react';
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';


const styles = theme => ({
});

class ComponentTemplate extends React.Component {
  state = {
  };

  render () {
    const {classes} = this.props;

    return(
      <div>
      </div>
    );
  }
}

ComponentTemplate.propTypes = {
  classes: PropTypes.object.isRequired,
};

export default withStyles(styles)(ComponentTemplate);
