# Narration (read any book aloud)

An optional module. Turn it on in **Settings → Modules → Narration**. With it on
and a voice runtime installed, the Readium reader shows a speaker button. It reads
the book out loud with a local [Pocket TTS](https://github.com/kyutai-labs/pocket-tts)
voice, sentence by sentence, marks the sentence being read, turns pages for you and
carries on into the next chapter. It reads from your own server; nothing is sent
to a cloud service.

Books that ship with recorded narration (EPUB 3 Media Overlays) keep their own
read-along button; this module is for every other book.

## How it works

- The server runs one small Python worker (`internal/narration/worker.py`) that
  keeps the model loaded. It starts when someone first presses play and is
  unloaded after ten idle minutes (`idle_seconds`). A server that never narrates
  never starts it, and a crash in the worker cannot take the server down.
- The reader splits the chapter into sentences, asks the server for the audio of
  the next three while the current one plays, and plays them back to back.
- Speed in the reader changes playback speed; the voice changes the speaker.
  Both are remembered in the browser.
- Requests are limited to about 1,500 characters, at most 8 wait at once, and
  only the built-in preset voices can be asked for.

## Install

Narration needs Python 3.13 with `pocket-tts` and a CPU build of PyTorch. The
runtime is not part of the normal image.

```sh
uv venv --python python3.13 /opt/bobobooks-voice
uv pip install --python /opt/bobobooks-voice/bin/python \
  torch==2.14.1+cpu --index-url https://download.pytorch.org/whl/cpu
uv pip install --python /opt/bobobooks-voice/bin/python pocket-tts==3.3.0 torchao==0.13.0
```

Then add to `config.json` and switch the module on in Settings:

```json
"narration": {
  "python": "/opt/bobobooks-voice/bin/python",
  "cache": "/opt/bobobooks-voice/hf-cache",
  "threads": 6,
  "offline": false,
  "token": ""
}
```

`cache` is where the model weights are kept (set `offline` to `true` once they
are downloaded to stop the worker contacting Hugging Face).

On Docker, the standard image is Alpine and cannot run PyTorch; use the narration
image variant instead (`compose.narration.yaml`).

## Voices

Settings → Modules → Narration voices lists the voices you made and is where new
ones are made: upload a recording, or record one in the browser. Use 3 to 30
seconds of clear, continuous speech (10 to 20 is a good start) and pick English or
Dutch, because a voice belongs to the language model that made it. The recording
stays on the server. Saved voices appear first in the reader's Read aloud menu,
and each can be tried, renamed or deleted from Settings.

Voices are folders (`voice.json`, `sample.wav`, `state.safetensors`) in
`<cache directory>/narration/voices` (`voice_directory` changes it). This is the
layout [Mochi](../../mochi2) writes, so Mochi's voices can be copied in as they are:
copy each folder's three files, then fix their owner to the user the server runs as.

Cloning needs the Pocket weights that include voice cloning, which Hugging Face
gates behind an access agreement. Either set `narration.token` to a read token from
an account that accepted it, or copy the downloaded `models--kyutai--pocket-tts`
folder into the cache and set `"offline": true`. Never copy the two 24-layer models
(`*24l*`); they are large and not used.

## Speed

On a 12-core CPU with 6 threads Pocket TTS speaks at about 1.5 times real time
(Mochi's earlier measurements and this module agree). The first sentence takes a
few seconds longer because the model has to load. With 2 threads it is about
1.25 times real time, which is enough but leaves little spare.

## Not yet

A sleep timer, narration in the classic reader, and reading documents aloud.
