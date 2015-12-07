import React from 'react'
import classNames from 'classnames'

import heimURL from '../../lib/heimURL'


export default function PolicyNav(props) {
  const items = [
    {name: 'values', caption: <span>Values</span>},
    {name: 'conduct', caption: <span><span className="long">Code of </span>Conduct</span>},
    {name: 'hosts', caption: <span><span className="long">Hosting </span>Rooms</span>},
    {name: 'terms', caption: <span>Terms<span className="long"> of Service</span></span>},
    {name: 'privacy', caption: <span>Privacy<span className="long"> Policy</span></span>},
  ]

  return (
    <nav>
      <div className="container">
        <span className="label">Platform Policies:</span>
        <ul>
          {items.map(item =>
            <li key={item.name} className={classNames(props.selected === item.name && 'selected')}>
              <a href={heimURL('/about/' + item.name)}>{item.caption}</a>
            </li>
          )}
        </ul>
      </div>
    </nav>
  )
}

PolicyNav.propTypes = {
  selected: React.PropTypes.string,
}
