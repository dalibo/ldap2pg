default:

tests:
	docker-compose up --abort-on-container-exit runner

clean:
	docker-compose down -v

debug:
	docker-compose exec runner /bin/bash
