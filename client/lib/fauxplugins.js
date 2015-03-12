var React = require('react')


module.exports = function(roomName) {
  if (roomName == 'thedrawingroom' || roomName == 'lovenest') {
    Heim.hook('page-bottom', function() {
      return (
        <style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
          .chat {
            background: #333;
          }

          .chat .room .name {
            color: #222;
          }

          .chat time {
            opacity: .5;
          }

          .chat .room .privacy-level,
          .nick {
            background: #e8e8e8 !important;
          }

          .message-emote {
            background: #f3f3f3 !important;
          }

          a {
            color: #444;
            text-decoration: none;
            font-weight: bold;
          }
        `}} />
      )
    })
  }

  if (roomName == 'space') {
    Heim.hook('sidebar', function() {
      return (
        <div key="norman" className="norman">
          <p>norman</p>
          <img src="//i.imgur.com/45wJkX7.jpg" />
        </div>
      )
    })

    Heim.hook('page-bottom', function() {
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
  }
}
