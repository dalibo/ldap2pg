default:

rpm:
	make -C ../../packaging rpm

tests:
	docker-compose up --remove-orphans --abort-on-container-exit runner

clean:
	make -C ../../packaging clean
	docker-compose down -v

debug:
	docker-compose exec runner /bin/bash
