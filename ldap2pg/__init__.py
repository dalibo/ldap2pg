from pkg_resources import get_distribution
import logging


class ChangeLogger(logging.Logger):
    def change(self, msg, *args, **kwargs):
        if self.isEnabledFor(logging.CHANGE):
            self._log(logging.CHANGE, msg, args, **kwargs)


logging.CHANGE = logging.INFO + 5
logging.addLevelName(logging.CHANGE, 'CHANGE')
logging.setLoggerClass(ChangeLogger)


__dist__ = get_distribution('ldap2pg')
__version__ = __dist__.version
