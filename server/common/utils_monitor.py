import threading
from collections import defaultdict
from . import utils

class UtilsMonitor:
    def __init__(self) -> None:
        self._lock = threading.Lock()

    def store_bets(self, bets: list[utils.Bet]) -> None:
        with self._lock:
            utils.store_bets(bets)
        
    def compute_winners(self) -> dict[int, list[int]]:
        winners_by_agency = defaultdict(list)
        with self._lock:
            for bet in utils.load_bets():
                if utils.has_won(bet):
                    try:
                        winners_by_agency[bet.agency].append(int(bet.document))
                    except Exception:
                        pass
        return winners_by_agency
