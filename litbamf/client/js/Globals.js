import bamflogo from './bamf.ascii.js';

class Navbar extends React.Component {
  constructor (props) {
    super(props);
  }

  render () {
    return (
      <nav>
        <pre>{bamflogo}</pre>
        <ul>
          <li><a href="/overview">Overview</a></li>
          <li><a href="/channels">Channels</a></li>
          <li><a href="/legacy">Legacy</a></li>
        </ul>
      </nav>
    );
  }
}

export {Navbar};