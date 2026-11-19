"""Bounded local speech recognition (faster-whisper, CPU). Nothing is logged: not the audio, not the words."""
import glob
import json
import os
import sys

MARK = 'NARRATION_TRANSCRIPT:'


def reply(payload):
    print(MARK + json.dumps(payload), flush=True)


try:
    args = json.loads(sys.stdin.read())
    cache = args['cache']
    os.environ['HF_HOME'] = cache
    os.environ['HF_HUB_OFFLINE'] = '1'
    os.environ['HF_HUB_DISABLE_IMPLICIT_TOKEN'] = '1'
    os.environ['OMP_NUM_THREADS'] = str(args['threads'])
    from faster_whisper import WhisperModel
    import av
    import numpy as np
    seconds_limit = 60
    # Decode incrementally so a compressed file cannot expand past the limit; container metadata alone is no bound.
    with av.open(args['input'], options={'protocol_whitelist': 'file',
                 'format_whitelist': 'wav,mp3,flac,ogg,mov,matroska,webm,aac',
                 'enable_drefs': '0'}) as container:
        resampler = av.AudioResampler(format='fltp', layout='mono', rate=16000)
        pieces, count = [], 0
        for frame in container.decode(audio=0):
            for part in resampler.resample(frame):
                count += part.samples
                if count > 16000 * seconds_limit:
                    raise ValueError('audio_too_long')
                pieces.append(part.to_ndarray().reshape(-1))
        for part in resampler.resample(None):
            count += part.samples
            if count > 16000 * seconds_limit:
                raise ValueError('audio_too_long')
            pieces.append(part.to_ndarray().reshape(-1))
    if count < 1600:
        raise ValueError('invalid_audio')
    audio = np.concatenate(pieces).astype(np.float32)
    if not np.isfinite(audio).all():
        raise ValueError('invalid_audio')
    snapshots = sorted(glob.glob(os.path.join(cache, 'whisper', 'models--Systran--faster-whisper-base.en', 'snapshots', '*')))
    if not snapshots:
        raise FileNotFoundError('model_missing')
    model = WhisperModel(snapshots[-1], device='cpu', compute_type='int8', cpu_threads=args['threads'], num_workers=1,
                         download_root=os.path.join(cache, 'whisper'), local_files_only=True)
    segments, _ = model.transcribe(audio, beam_size=3, language='en', vad_filter=True, condition_on_previous_text=False)
    text = ' '.join(segment.text.strip() for segment in segments).strip()
    if not text:
        raise ValueError('no_speech')
    reply({'ok': True, 'text': text[:6000]})
except Exception as error:
    code = 'failed'
    if isinstance(error, (ImportError, ModuleNotFoundError, FileNotFoundError)):
        code = 'unavailable'
    elif str(error) in ('audio_too_long', 'invalid_audio', 'no_speech'):
        code = str(error)
    reply({'ok': False, 'code': code})
    sys.exit(1)
