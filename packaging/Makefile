default:

distclean:
	rm -rf ../dist/

rpm:
	docker-compose run --rm rpm
# building rpm update egg_info for python2.7. We need to regenerate it for
# development.
	cd ../ && python setup.py egg_info

clean:
	docker-compose down -v
