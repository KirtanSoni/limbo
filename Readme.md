# DnD with AI. 
play dnd with AI as your game master.
 
## Start Instructions

### Makefile has all instructions
```
make 
./limbo  or ./limbo.exe
```
### React Dev Mode ( for Hot-Reload )
```
make fdev
```

### #TODO
comments with TODO require Attention 

## Strategy

- first we should build a chat app that an ai is able to participate in. and test it out.
![alt text](/docs/img/image.png)



Using WebRTC - SFU (Selective forwarding unit)

Official Example : https://github.com/pion/example-webrtc-applications/tree/master/sfu-ws

Community made Simple SFU : https://github.com/tiger2380/simple_sfu/tree/master

## Game Functionalities:

ServerSide: 

- Join and Leave Room. 
- Voice chat and realtime transcription
- Server LLM interaction, Hopefully Speech return.
- Some Game State Management and forwarding. ( mostly using websockets. )
- no Text Chat for now. 

MVP: one client connects to server and is able to talk to an LLM , llm replies through text using websockets.

