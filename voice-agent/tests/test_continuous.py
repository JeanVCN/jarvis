import asyncio
import os
from fractions import Fraction
import numpy as np
import pytest
from jarvis_voice_agent.continuous import OutputTrack, sentences
from jarvis_voice_agent.engines import Profile


def test_sentences_handle_decimal_and_bound_unpunctuated_text():
    parts,rest=sentences("O valor é 25.50 reais. Próxima")
    assert parts==["O valor é 25.50 reais."]
    assert rest=="Próxima"
    parts,rest=sentences("palavra "*100)
    assert parts and max(map(len,parts))<=180
    assert len(rest)<=180


def test_profiles_do_not_enable_synthetic_audio_by_accident(monkeypatch):
    monkeypatch.delenv("JARVIS_TEST_AUDIO",raising=False)
    with pytest.raises(ValueError,match="synthetic_audio_disabled"):
        Profile(stt="fixture",tts="tone").validate()
    with pytest.raises(ValueError,match="invalid_pocket_voice"):
        Profile(tts="pocket",voice="pf_dora").validate()


def test_interrupt_drops_queued_audio_and_late_generation():
    async def run():
        events=[]
        track=OutputTrack(lambda *e:events.append(e))
        meta={"turn_id":"a","segment":0}
        await track.put(np.ones(480,np.float32)*.1,24000,0,meta)
        track.clear()
        await track.put(np.ones(480,np.float32)*.1,24000,0,meta)
        frame=await track.recv()
        assert not frame.to_ndarray().any()
        assert not events
        await track.put(np.ones(480,np.float32)*.1,24000,1,{"turn_id":"b","segment":0})
        frame=await track.recv()
        assert frame.to_ndarray().any()
        assert events[0][1]["turn_id"]=="b"
        track.stop()
    asyncio.run(run())


def test_silero_and_smart_turn_load_and_process_silence():
    from pipecat.audio.vad.silero import SileroVADAnalyzer
    from pipecat.audio.turn.smart_turn.local_smart_turn_v3 import LocalSmartTurnAnalyzerV3
    from pipecat.audio.vad.vad_analyzer import VADState
    async def run():
        vad=SileroVADAnalyzer(sample_rate=16000)
        vad.set_sample_rate(16000)
        assert await vad.analyze_audio(bytes(1024))==VADState.QUIET
        smart=LocalSmartTurnAnalyzerV3(sample_rate=16000,cpu_count=1)
        smart.set_sample_rate(16000)
        smart.append_audio(bytes(32000),False)
        state,metrics=await smart.analyze_end_of_turn()
        assert state is not None
        await smart.cleanup()
    asyncio.run(run())
