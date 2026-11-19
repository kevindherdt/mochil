"""Narration worker: keeps one Pocket TTS model loaded and speaks one sentence per request.

Protocol: one JSON object per line on stdin, one per line on stdout. The real
stdout is reserved for replies; anything a library prints goes to stderr.
  speak:    {"id": 1, "text": "...", "voice": "alba" | "/path/state.safetensors", "language": "english" | "dutch"}
  warm:     {"id": 3, "op": "warm", "voice": "alba", "language": "english"}  loads the model and voice, says nothing
  encode:   {"id": 2, "op": "encode", "reference": "/path/reference.wav", "state": "/path/state.safetensors", "language": "english"}
  reply:    {"id": 1, "ok": true, "sample_rate": 24000, "pcm": "<base64 int16 LE>", "seconds": 1.2, "took": 0.5}
            {"id": 1, "ok": false, "code": "..."}
A first line {"ready": true} is sent once the libraries are loaded; each model loads on first use.
"""
import base64
import json
import os
import re
import sys
import time
import unicodedata

reply_stream = sys.stdout
sys.stdout = sys.stderr

MAX_TEXT = 1200
MAX_VOICES = 6
MAX_MODELS = 2


def speakable(text):
    """One sentence of book text, made safe for a model trained on punctuated sentences."""
    text = ''.join(ch for ch in text if unicodedata.category(ch) not in ('So', 'Cs', 'Co', 'Cc') or ch in '\n\t')
    text = re.sub(r'\s*-{2,}\s*|\s+[—–-]\s+|—|–', ', ', text)
    text = re.sub(r'[*_`#~|>\[\]{}]+', ' ', text)
    text = re.sub(r'\s+', ' ', text).strip(' ,;')
    if not re.search(r'\w', text):
        return ''
    letters = re.sub(r'[^A-Za-z]', '', text)
    if len(letters) >= 4 and letters.isupper() and len(text.split()) >= 2:
        # A heading in capitals is read as a heading, not spelled out letter by letter.
        text = text.lower().capitalize()
    core = text.rstrip('"\'”’»)')
    if core and core[-1] in ':;,':
        text = core[:-1].rstrip() + '.' + text[len(core):]
    elif core and core[-1] not in '.!?…':
        text += '.'
    return text


def send(value):
    reply_stream.write(json.dumps(value) + '\n')
    reply_stream.flush()


def main():
    config = json.loads(sys.argv[1]) if len(sys.argv) > 1 else {}
    os.environ['HF_HOME'] = config.get('cache') or os.path.expanduser('~/.cache/huggingface')
    os.environ['OMP_NUM_THREADS'] = str(config.get('threads', 2))
    os.environ['OPENBLAS_NUM_THREADS'] = '1'
    os.environ['HF_HUB_DISABLE_IMPLICIT_TOKEN'] = '1'
    if config.get('offline'):
        os.environ['HF_HUB_OFFLINE'] = '1'
    if config.get('token'):
        os.environ['HF_TOKEN'] = config['token']
        os.environ['HF_HUB_DISABLE_IMPLICIT_TOKEN'] = '0'
    started = time.perf_counter()
    import numpy as np
    import torch
    torch.set_num_threads(config.get('threads', 2))
    torch.set_num_interop_threads(1)
    from pocket_tts import TTSModel, export_model_state
    torch.set_num_threads(config.get('threads', 2))  # importing pocket_tts resets the count

    models = {}   # language -> model; a voice belongs to the model that made it
    voices = {}   # (language, voice) -> prepared voice state

    def model_for(language):
        if language not in models:
            if len(models) >= MAX_MODELS:
                oldest = next(iter(models))
                models.pop(oldest)
                for key in [key for key in voices if key[0] == oldest]:
                    voices.pop(key)
            models[language] = TTSModel.load_model(language=language, quantize=config.get('quantize', True),
                                                   temp=config.get('temperature', 0.3))
        return models[language]

    send({'ready': True, 'load_seconds': time.perf_counter() - started})

    for line in sys.stdin:
        try:
            request = json.loads(line)
        except ValueError:
            continue
        identifier = request.get('id')
        began = time.perf_counter()
        language = request.get('language') if request.get('language') in ('english', 'dutch') else 'english'
        try:
            model = model_for(language)
            if request.get('op') == 'warm':
                key = str(request.get('voice') or 'alba')
                if (language, key) not in voices:
                    if len(voices) >= MAX_VOICES:
                        voices.pop(next(iter(voices)))
                    voices[(language, key)] = model.get_state_for_audio_prompt(key)
                send({'id': identifier, 'ok': True, 'took': time.perf_counter() - began})
                continue
            if request.get('op') == 'encode':
                # Turn a prepared recording into a saved voice. Needs the model that can clone.
                if not model.has_voice_cloning:
                    send({'id': identifier, 'ok': False, 'code': 'cloning_unavailable'})
                    continue
                state = model.get_state_for_audio_prompt(request['reference'])
                export_model_state(state, request['state'])
                send({'id': identifier, 'ok': True, 'took': time.perf_counter() - began})
                continue
            text = speakable(str(request.get('text', ''))[:MAX_TEXT])
            key = str(request.get('voice') or 'alba')
            if (language, key) not in voices:
                if len(voices) >= MAX_VOICES:
                    voices.pop(next(iter(voices)))
                voices[(language, key)] = model.get_state_for_audio_prompt(key)
            if not text:
                send({'id': identifier, 'ok': True, 'sample_rate': model.sample_rate, 'pcm': '', 'seconds': 0, 'took': 0})
                continue
            if request.get('stream'):
                # Send each piece as it is made, so playback can start after the first one.
                total = 0
                for chunk in model.generate_audio_stream(voices[(language, key)], text, copy_state=True):
                    part = (chunk.detach().cpu().numpy() if hasattr(chunk, 'detach') else np.asarray(chunk)).reshape(-1)
                    if not len(part) or not np.isfinite(part).all():
                        raise ValueError('invalid_audio')
                    total += len(part)
                    pcm = (np.clip(part, -1, 1) * 32767).astype('<i2').tobytes()
                    send({'id': identifier, 'more': True, 'sample_rate': model.sample_rate, 'pcm': base64.b64encode(pcm).decode('ascii')})
                if not total:
                    raise ValueError('invalid_audio')
                send({'id': identifier, 'ok': True, 'sample_rate': model.sample_rate, 'seconds': total / model.sample_rate, 'took': time.perf_counter() - began})
                continue
            parts = []
            for chunk in model.generate_audio_stream(voices[(language, key)], text, copy_state=True):
                part = chunk.detach().cpu().numpy() if hasattr(chunk, 'detach') else np.asarray(chunk)
                parts.append(part.reshape(-1))
            audio = np.concatenate(parts) if parts else np.zeros(0, dtype=np.float32)
            if not len(audio) or not np.isfinite(audio).all():
                raise ValueError('invalid_audio')
            pcm = (np.clip(audio, -1, 1) * 32767).astype('<i2').tobytes()
            send({'id': identifier, 'ok': True, 'sample_rate': model.sample_rate,
                  'pcm': base64.b64encode(pcm).decode('ascii'),
                  'seconds': len(audio) / model.sample_rate, 'took': time.perf_counter() - began})
        except Exception as error:  # never echo exception text back to a client
            print('narration worker error:', repr(error)[:300], file=sys.stderr, flush=True)
            send({'id': identifier, 'ok': False, 'code': 'generation_failed'})


if __name__ == '__main__':
    main()
