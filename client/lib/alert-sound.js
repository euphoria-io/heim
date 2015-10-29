const sound = new Audio(process.env.HEIM_PREFIX + '/static/alert.mp3')

export function play() {
  if (sound.readyState !== 0) {
    sound.currentTime = 0
  }
  sound.play()
}
