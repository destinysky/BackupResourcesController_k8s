from socketserver import BaseRequestHandler, TCPServer

class EchoHandler(BaseRequestHandler):
    def handle(self):
        print('Got connection from', self.client_address)
        while True:
            msg = str(self.request.recv(8192),encoding='utf-8')
            print(msg)
            if not msg or msg=="ok":
                print("exit")
                self.request.send(bytes("bye",encoding='utf-8'))
                exit()
            

if __name__ == '__main__':
    serv = TCPServer(('', 20000), EchoHandler)
    serv.serve_forever()