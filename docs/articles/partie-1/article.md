On utilise des serveurs web tous les jours sans jamais les voir. Nginx, Spring Boot, Express, Django : tous cachent la même mécanique sous plusieurs couches d'abstraction. Dans cette série, on la remet à nu en construisant **Kross**, un serveur web écrit en Go, en partant des sockets TCP.

Le but n'est pas de concurrencer Nginx. Le but est de comprendre ce qui se passe entre le moment où ton navigateur tape une adresse et celui où ton code reçoit une requête. Une fois que tu l'as construit toi-même, un « address already in use », un « connection reset by peer » ou un timeout ne sont plus des messages mystérieux : ce sont des étapes que tu reconnais.

![Kross, construire un serveur web from scratch](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/cover.png "Une connexion, c'est une passe : il faut quelqu'un pour la recevoir.")

## Pourquoi « Kross » ?

Pendant dix saisons au Real Madrid, Toni Kroos a tenu le milieu de terrain avec une spécialité : recevoir le ballon, lever la tête, et le rendre au bon endroit, au bon moment. Réputé pour la précision de ses passes, il perdait rarement le ballon et faisait rarement de bruit.

Un serveur web fait exactement ce travail. Il reçoit une requête, comprend ce qu'on lui demande, et renvoie la bonne réponse sans faire tomber le jeu. Le nom était trouvé : **Kross**, un clin d'œil au numéro 8 du Real, avec un K pour la touche de caractère.

> [!NOTE] Dans cette première partie, Kross ne fait encore qu'une chose : il ouvre la porte et accueille les connexions. Il apprendra à lire les passes, c'est-à-dire les requêtes HTTP, dans la partie 2.

## Un serveur web, c'est quoi exactement ?

Un serveur web est un programme qui attend. Il s'installe sur une adresse et un port, par exemple `127.0.0.1:8080`, et patiente jusqu'à ce qu'un client vienne frapper. Quand un client arrive, ils ouvrent ensemble une **connexion TCP**, un tuyau fiable dans lequel les octets arrivent dans l'ordre et sans perte. Dans ce tuyau, le client écrit sa requête ; le serveur la lit, la traite, puis écrit sa réponse.

![Les couches d'un serveur web : HTTP au-dessus de TCP, au-dessus d'IP](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/layers.png "HTTP voyage dans TCP, qui voyage dans IP. La partie 1 s'occupe de TCP.")

Ce qui surprend souvent, c'est que HTTP n'est que du texte. Voici, mot pour mot, ce que `curl` envoie quand tu tapes `curl http://127.0.0.1:8080/` :

```http
GET / HTTP/1.1
Host: 127.0.0.1:8080
User-Agent: curl/8.5.0
Accept: */*
```

Une ligne de requête, quelques en-têtes, une ligne vide. Rien de magique. Mais pour lire ces quatre lignes, il faut d'abord une connexion ouverte. C'est tout l'objet de cette première partie.

## Le choix du langage : pourquoi Go ?

Avant d'écrire une ligne, il fallait choisir. Voici le raisonnement, langage par langage :

- **C** : c'est le langage des appels système, au plus près du noyau. Mais la gestion manuelle de la mémoire aurait transformé la série en cours de C.
- **Rust** : sûr et rapide, mais le borrow checker et l'async ajoutent une marche importante avant même de parler réseau.
- **Python** : très lisible, mais il cache presque tout, et les performances deviennent vite le sujet.
- **Java** : solide, mais accéder directement à `socket`, `bind` et `listen` demande de sortir de la bibliothèque standard.
- **Go** : le paquet `syscall` donne accès aux appels système sans détour, les goroutines rendront la concurrence simple quand on en aura besoin, et le résultat est un seul binaire facile à lancer.

Go l'emporte : assez bas niveau pour voir chaque étape, assez simple pour rester lisible, quel que soit ton langage de départ.

> [!WARNING] Règle du jeu pour toute la série : on s'interdit le paquet `net/http`. Le but est justement de l'écrire.

## Ce que nous allons construire

Voici la feuille de route envisagée. Elle pourra évoluer au fil des parties, mais elle donne le cap :

1. **Partie 1** : accepter des connexions TCP (cet article)
2. **Partie 2** : lire et comprendre une requête HTTP
3. **Partie 3** : répondre, avec un code de statut, des en-têtes et un corps
4. **Partie 4** : servir plusieurs clients à la fois avec les goroutines
5. **Partie 5** : router les requêtes et servir des fichiers

Pour cette première étape, la définition de « terminé » tient en une phrase : **Kross démarre sur une adresse, accepte n'importe quelle connexion, la referme proprement, et s'arrête sans bavure quand on appuie sur Ctrl+C**. Le tout couvert par des tests.

## Le vocabulaire minimum

Quatre mots reviennent sans cesse quand on parle réseau au niveau du système :

- **Le noyau** : le cœur du système d'exploitation, Linux ou macOS. C'est lui qui parle réellement à la carte réseau. Ton programme lui demande des services.
- **Le socket** : une extrémité de connexion, gérée par le noyau. Ton programme ne le touche jamais directement.
- **Le descripteur de fichier** : le simple entier que le noyau te donne pour désigner ce socket. Sur Unix, presque tout est un fichier, y compris une connexion réseau.
- **L'adresse et le port** : l'adresse IP désigne la machine, le port désigne le programme. `127.0.0.1` est ta propre machine, et le port est un nombre entre 0 et 65535.

## Les quatre appels système

Accepter une connexion demande exactement quatre appels au noyau, toujours dans le même ordre :

1. `socket` : « crée-moi une extrémité de connexion TCP ». Le noyau répond par un descripteur.
2. `bind` : « attache ce socket à telle adresse et tel port ».
3. `listen` : « commence à écouter, et garde les clients qui arrivent dans une file d'attente ».
4. `accept` : « donne-moi le prochain client de la file ». S'il n'y en a pas, on attend.

![Diagramme de séquence : socket, bind, listen, accept, puis la poignée de main TCP](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/syscalls.png "La poignée de main TCP (SYN, SYN-ACK, ACK) se fait entre le client et le noyau, avant même que Kross appelle accept().")

Le détail le plus important du schéma est en bas : la fameuse poignée de main TCP en trois temps n'implique pas ton programme. Le noyau la termine seul, puis range la connexion prête dans la file d'attente. `accept` ne fait que venir la chercher.

> [!TIP] C'est pour ça qu'un client peut se connecter à un serveur « occupé » : la connexion est établie par le noyau et attend dans la file, même si le programme n'a pas encore appelé `accept`.

### Pourquoi ne pas simplement appeler net.Listen ?

En Go, `net.Listen("tcp", ":8080")` enchaîne `socket`, `bind` et `listen` en une seule ligne. C'est parfait en production, mais pour apprendre, ça cache tout. Et surtout, Go choisit lui-même la taille de la file d'attente à partir d'un réglage du système (sur Linux, `/proc/sys/net/core/somaxconn`). Kross veut décider de cette taille, et voir chaque étape passer. On fait donc les appels à la main.

## Le code, pièce par pièce

Avant d'entrer dans les fichiers, voici comment Kross est découpé :

![Architecture de Kross : main, Server, l'interface Listener, TcpListener, ClientConnection et ServerConfig](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/architecture.png "Chaque paquet a une seule responsabilité. Server ne dépend que d'une interface.")

### 1. La configuration

Trois réglages suffisent pour démarrer : l'adresse, le port et la taille de la file d'attente (le **backlog**).

```go
type ServerConfig struct {
	Host    string
	Port    int
	Backlog int
}

func Default() ServerConfig {
	return ServerConfig{
		Host:    "127.0.0.1",
		Port:    8080,
		Backlog: 128,
	}
}

func (c ServerConfig) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	if c.Backlog < 1 {
		return fmt.Errorf("backlog must be at least 1: %d", c.Backlog)
	}
	return nil
}
```

Le port tient sur 16 bits, d'où la borne à 65535. Le port 0 est volontairement accepté : il demande au noyau de choisir un port libre, ce qui sera très pratique dans les tests.

### 2. Bind : créer le socket et lui donner une adresse

```go
func (l *TcpListener) Bind() error {
	ip := net.ParseIP(l.host).To4()
	if ip == nil {
		return fmt.Errorf("invalid IPv4 address: %q", l.host)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	// Allows restarting without waiting for TIME_WAIT to expire.
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("setsockopt: %w", err)
	}

	sa := &syscall.SockaddrInet4{Port: l.port}
	copy(sa.Addr[:], ip)

	if err := syscall.Bind(fd, sa); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("bind %s:%d: %w", l.host, l.port, err)
	}

	l.fd = fd
	return nil
}
```

Trois choses à remarquer :

- `AF_INET` veut dire IPv4, et `SOCK_STREAM` veut dire TCP. Le résultat, `fd`, est le fameux descripteur : un simple entier.
- À chaque erreur, le descripteur est refermé avant de sortir. Sinon, chaque démarrage raté laisserait traîner un socket ouvert.
- L'option `SO_REUSEADDR` mérite son propre encadré.

> [!WARNING] Sans `SO_REUSEADDR`, si tu arrêtes Kross et le relances aussitôt, tu obtiens souvent « bind: address already in use ». Après une fermeture, le noyau garde l'adresse en réserve un moment, dans l'état TIME_WAIT, pour absorber les derniers paquets en retard. Cette option l'autorise à réutiliser l'adresse tout de suite.

### 3. Listen : ouvrir la porte et installer la file d'attente

```go
func (l *TcpListener) Listen() error {
	if l.fd < 0 {
		return errors.New("listen called before bind")
	}

	if err := syscall.Listen(l.fd, l.backlog); err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	// os.NewFile takes ownership of the descriptor.
	f := os.NewFile(uintptr(l.fd), "tcp-listener")
	l.fd = -1

	ln, err := net.FileListener(f)
	f.Close() // net.FileListener duplicates the descriptor, so f can be closed
	if err != nil {
		return fmt.Errorf("convert to net.Listener: %w", err)
	}

	l.ln = ln
	return nil
}
```

C'est ici que le backlog entre en jeu. C'est la taille de la file où le noyau range les connexions déjà établies, en attendant que Kross vienne les chercher avec `accept`.

![Le backlog : des clients arrivent, le noyau les range dans une file, Kross les prend une par une](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/backlog.png "Si la file déborde, les nouveaux clients patientent ou échouent. Avec 128 places, Kross a de la marge.")

La deuxième moitié de la fonction est la partie la plus subtile de Kross. Une fois le socket en écoute, on le confie au runtime de Go avec `os.NewFile` puis `net.FileListener`. Pourquoi ne pas continuer avec `syscall.Accept` ? Parce que le runtime de Go sait attendre efficacement : avec epoll sur Linux ou kqueue sur macOS, une goroutine qui attend un client ne bloque pas un thread du système. On a fait nous-mêmes tout ce qui comptait pour comprendre, puis on passe la main à la mécanique qui fait bien son travail.

> [!TIP] Tu peux voir la file d'attente de tes propres yeux. Lance Kross, puis tape `ss -ltn` (sur macOS : `netstat -an | grep LISTEN`). Pour un socket en écoute, la colonne Send-Q affiche la taille du backlog, et Recv-Q le nombre de connexions qui attendent.

Voici ce que donne `ss` sur Kross lancé sur le port 8099 :

```bash
$ ss -ltn 'sport = :8099'
State  Recv-Q Send-Q Local Address:Port Peer Address:Port
LISTEN 0      128        127.0.0.1:8099      0.0.0.0:*
```

Les 128 places sont là, et aucune connexion n'attend.

### 4. Accept : recevoir un client

```go
func (l *TcpListener) Accept() (*conn.ClientConnection, error) {
	if l.ln == nil {
		return nil, errors.New("accept called before listen")
	}

	raw, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}
	return conn.New(raw), nil
}
```

Chaque client accepté est emballé dans une `ClientConnection`, un type à nous qui sait lire, écrire et fermer, comme n'importe quel `io.ReadWriteCloser` de Go :

```go
type ClientConnection struct {
	raw net.Conn
}

func (c *ClientConnection) Read(p []byte) (int, error) {
	return c.raw.Read(p)
}

func (c *ClientConnection) Write(p []byte) (int, error) {
	return c.raw.Write(p)
}
```

Aujourd'hui, ce type ne fait que transmettre. Mais c'est lui qui accueillera plus tard les délais de lecture, la mise en mémoire tampon ou les statistiques, sans que le reste du serveur ait à changer.

### 5. Le serveur : orchestrer le tout

```go
func (s *Server) Start() error {
	if err := s.listener.Bind(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	if err := s.listener.Listen(); err != nil {
		s.listener.Close()
		return fmt.Errorf("start: %w", err)
	}
	return s.acceptLoop()
}

func (s *Server) acceptLoop() error {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		s.handle(c)
	}
}

func (s *Server) handle(c *conn.ClientConnection) {
	defer c.Close()
}
```

Le cœur de Kross tient dans cette boucle : accepter, traiter, recommencer. Pour l'instant, « traiter » veut dire refermer aussitôt la connexion. C'est voulu, et on va voir juste après ce que ça provoque côté client.

Un détail compte beaucoup pour la suite : `Server` ne connaît pas `TcpListener`, seulement l'interface `Listener`. Dans les tests, on peut donc lui donner un faux listener, sans aucun réseau.

### 6. main : configuration, signaux et arrêt propre

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

l := listener.NewTcpListener(cfg.Host, cfg.Port, cfg.Backlog)
srv := server.New(cfg, l)

go func() {
	<-ctx.Done()
	if err := srv.Stop(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}()

log.Printf("listening on %s:%d (backlog %d)", cfg.Host, cfg.Port, cfg.Backlog)
if err := srv.Start(); err != nil {
	log.Fatalf("server: %v", err)
}
log.Println("server stopped")
```

`Start` bloque dans `Accept` tant que personne ne frappe. Pour arrêter Kross proprement, une goroutine attend le signal Ctrl+C, puis ferme le socket d'écoute. Fermer ce socket débloque `Accept`, qui renvoie `net.ErrClosed`, et la boucle comprend que c'est un arrêt normal :

![Diagramme de séquence de l'arrêt : Ctrl+C, Stop, Close, Accept renvoie net.ErrClosed, Start renvoie nil](https://raw.githubusercontent.com/brandonkamga237/kross/master/docs/articles/partie-1/images/shutdown.png "Aucune connexion n'est coupée brutalement : le serveur sort de sa boucle par la grande porte.")

## On l'essaie

```bash
git clone https://github.com/brandonkamga237/kross.git
cd kross
make build
./bin/webserver -port 8099
```

Kross s'annonce :

```bash
2026/10/10 01:46:53 listening on 127.0.0.1:8099 (backlog 128)
```

Dans un deuxième terminal, on frappe à la porte avec `nc` :

```bash
$ nc -v 127.0.0.1 8099
Connection to 127.0.0.1 8099 port [tcp/*] succeeded!
```

La connexion est acceptée. Maintenant, essayons avec `curl`, qui envoie une vraie requête HTTP :

```bash
$ curl -v http://127.0.0.1:8099/
> GET / HTTP/1.1
> Host: 127.0.0.1:8099
> User-Agent: curl/8.5.0
> Accept: */*
>
* Recv failure: Connection reset by peer
curl: (56) Recv failure: Connection reset by peer
```

« Connection reset by peer » : Kross a raccroché au nez de curl. Kross a fermé la connexion sans lire la requête que curl venait d'envoyer. Quand un programme ferme un socket alors que des données non lues attendent encore dans son tampon de réception, le noyau ne fait pas une fermeture polie (FIN). Il coupe net, avec un paquet RST.

> [!NOTE] Ce n'est pas un bug, c'est le point de départ de la partie 2 : lire ce que le client envoie avant de répondre. Ce message disparaîtra dès que Kross lira les requêtes.

De retour dans le premier terminal, un Ctrl+C, et Kross sort proprement :

```bash
2026/10/10 01:46:54 server stopped
```

## Comment on teste

Un serveur réseau se teste mal « à la main ». Kross a donc ses tests dès le premier jour, lancés avec `make test` et le détecteur de concurrence de Go (`-race`) :

- **Le listener** démarre sur le port 0, laisse le noyau choisir un port libre, s'y connecte avec `net.Dial`, et vérifie qu'`Accept` renvoie bien un client. Deux autres tests vérifient qu'on ne peut pas écouter avant `bind`, ni s'attacher à une adresse invalide.
- **Le serveur** reçoit un faux listener qui enregistre les appels. Le test vérifie que `Bind` et `Listen` arrivent bien avant `Accept`, et que `Stop` fait revenir `Start` sans erreur.
- **La connexion** est testée avec `net.Pipe`, un faux tuyau en mémoire : ce qui est écrit d'un côté doit arriver intact de l'autre.

Voici le faux listener, qui montre bien l'intérêt de l'interface :

```go
func (f *fakeListener) Accept() (*conn.ClientConnection, error) {
	<-f.closed
	return nil, net.ErrClosed
}
```

Il bloque comme un vrai `Accept`, jusqu'à ce qu'on le ferme, puis renvoie exactement l'erreur du vrai. Le serveur ne voit pas la différence.

## Ce qu'il faut retenir

- Un serveur web commence par un socket : `socket`, `bind`, `listen`, puis `accept` en boucle.
- La poignée de main TCP est l'affaire du noyau. Ton programme récupère des connexions déjà établies.
- Le backlog est la file d'attente devant la porte. Kross en choisit la taille, et `ss -ltn` permet de la voir.
- `SO_REUSEADDR` évite le « address already in use » au redémarrage.
- Fermer une connexion sans lire ce qu'elle contient provoque un « connection reset by peer ».
- Programmer contre une interface (`Listener`) rend le serveur testable sans réseau.

## Les limites assumées de cette version

Kross est volontairement minimal à ce stade :

- il traite les connexions une par une, ce qui ne pose aucun problème tant qu'il les referme aussitôt (la partie 4 s'en occupera) ;
- il ne parle qu'IPv4 ;
- il fonctionne sur Linux et macOS (sous Windows, passe par WSL) ;
- et il ne comprend pas encore un mot de HTTP.

## Pour aller plus loin

[socket(2), la page de manuel Linux](https://man7.org/linux/man-pages/man2/socket.2.html)

[listen(2), et ce que dit vraiment le backlog](https://man7.org/linux/man-pages/man2/listen.2.html)

[accept(2), la page de manuel Linux](https://man7.org/linux/man-pages/man2/accept.2.html)

[Beej's Guide to Network Programming](https://beej.us/guide/bgnet/)

[RFC 9293, la spécification de TCP](https://www.rfc-editor.org/rfc/rfc9293)

[Le paquet syscall de Go](https://pkg.go.dev/syscall)

[Le code de Kross sur GitHub](https://github.com/brandonkamga237/kross)

## Dans la partie 2

Kross ouvre la porte, mais il ne laisse encore personne parler. Dans la prochaine partie, on lira enfin les octets que curl envoie : la ligne de requête, les en-têtes, la ligne vide qui marque la fin. On découpera tout ça proprement, et le « connection reset by peer » laissera place à une vraie conversation.
