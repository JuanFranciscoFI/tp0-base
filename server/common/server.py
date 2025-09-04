import socket
import logging
import signal
from common.protocol import Protocol
from common.utils import store_bets

class Server:
    def __init__(self, port, listen_backlog, agency_id=0):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._shutdown = False
        self._agency_id = agency_id

    def __graceful_shutdown(self, signum, frame):
        logging.info("action: graceful_shutdown | result: in_progress")
        self._shutdown = True
        self._server_socket.close()
        logging.info("action: graceful_shutdown | result: success")

    def run(self):
        signal.signal(signal.SIGINT, self.__graceful_shutdown)
        signal.signal(signal.SIGTERM, self.__graceful_shutdown)

        try:
            while not self._shutdown:
                try:
                    client_sock = self.__accept_new_connection()
                    if client_sock:
                        self.__handle_client_connection(client_sock)
                except OSError as e:
                    if not self._shutdown:
                        logging.error(f"action: accept_connection | error: {str(e)}")
        finally:
            self._server_socket.close()

    def __handle_client_connection(self, client_sock):
        proto = Protocol(client_sock, self._agency_id)
        try:
            bet = proto.recv_bet()
            store_bets([bet])
            proto.send_response(ok=True)
            logging.info(
                f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}"
            )

        except Exception as e:
            logging.error(f"action: apuesta_almacenada | result: fail | error: {e}")
            proto.send_response(ok=False)
           
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
