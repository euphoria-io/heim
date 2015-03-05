var React = require('react')


Heim.hook('page-bottom', function() {
  if (this.state.chat.roomName != 'thedrawingroom') {
    return
  }

  return (
    <style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
      .nick {
        background: #e8e8e8 !important;
      }
    `}} />
  )
})

Heim.hook('sidebar', function() {
  if (this.props.roomName != 'space') {
    return
  }

  return (
    <div key="norman" className="norman">
      <p>norman</p>
      <img src="//i.imgur.com/45wJkX7.jpg" />
    </div>
  )
})

Heim.hook('page-bottom', function() {
  if (this.state.chat.roomName != 'space') {
    return
  }

  return (
    <style key="norman-style" dangerouslySetInnerHTML={{__html:`
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
    `}} />
  )
})
