import React from 'react';


/*
 * Abstract class for our standard popup dialogs
 * See ChannelPayDialog for use example
 */
class PopUpDialog extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      open: false
    };
  }

  // override this method to reset any states that get set by handleChange
  resetState(state) {
  }

  handleClickOpen() {
    this.setState({open: true});
  };

  handleClose() {
    this.setState({open: false});
    this.resetState();
  };

  handleSubmit() {
    this.setState({open: false});
    this.resetState();
  };

  handleChange(name) {
    return (event => {
      this.setState({
        [name]: event.target.value,
      });
    });
  }
}


export default PopUpDialog;
