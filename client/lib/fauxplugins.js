var React = require('react')


Heim.plugins.hooks.pageBottom.listen(function(results, props, state) {
  if (state.chat.roomName == 'thedrawingroom') {
    results.push(<style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
      .nick {
        background: #e8e8e8 !important;
      }
    `}} />)
  }
})

Heim.plugins.hooks.sidebar.listen(function(results, props, state) {
  if (props.roomName == 'space') {
    results.push(
      <div key="norman" className="norman">
        <p>norman</p>
        <img src="//i.imgur.com/45wJkX7.jpg" />
      </div>
    )
  }
})

Heim.plugins.hooks.pageBottom.listen(function(results, props, state) {
  if (state.chat.roomName == 'space') {
    results.push(<style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
      .norman {
        margin-top: 15px;
        text-align: right;
        opacity: .5;
      }

      .norman, .norman img {
        transition: all .15s ease;
      }

      .norman:hover {
        opacity: 1;
      }

      .norman:hover img {
        width: 22vw;
        max-width: 250px;
      }

      .norman p {
        margin: 0;
        font-size: 12px;
      }

      .norman img {
        width: 15vw;
        min-width: 50px;
        max-width: 100px;
      }
    `}} />)
  }
})
