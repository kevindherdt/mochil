// Runs on the audio thread: hands the microphone to the page in blocks of 20 ms.
class HermesCapture extends AudioWorkletProcessor {
  constructor() {
    super();
    this.size = Math.max(128, Math.round(sampleRate * 0.02));
    this.block = new Float32Array(this.size);
    this.filled = 0;
  }
  process(inputs) {
    const channel = inputs[0] && inputs[0][0];
    if (channel) {
      for (let i = 0; i < channel.length; i += 1) {
        this.block[this.filled++] = channel[i];
        if (this.filled === this.size) { this.port.postMessage(this.block.slice()); this.filled = 0; }
      }
    }
    return true;
  }
}
registerProcessor("hermes-capture", HermesCapture);
