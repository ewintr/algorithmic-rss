
tui-deploy:
	go build -o rss-tui ./tui/...
	scp rss-tui server:dist
	rm rss-tui

service-deploy:
	go build -o algorithmic-rss ./service/...
	scp algorithmic-rss server:
	ssh server sudo systemctl stop algorithmic-rss.service
	ssh server sudo mv algorithmic-rss /usr/local/bin/algorithmic-rss
	ssh server sudo systemctl start algorithmic-rss.service
	rm algorithmic-rss

cli-build:
	go build -o algorithmic-rss-cli ./cli/...
