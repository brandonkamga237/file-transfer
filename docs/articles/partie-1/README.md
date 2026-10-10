# Partie 1 : Kross ouvre la porte

Article publié dans la bibliothèque LesCracks. Titre, description, catégorie, tags et couverture sont dans `meta.json`.

Pour le publier, depuis la racine du dépôt :

```bash
node docs/articles/publish.mjs docs/articles/partie-1            # en brouillon, à relire dans le back office
node docs/articles/publish.mjs docs/articles/partie-1 --publish  # en ligne
```

Le script demande l'identifiant et le mot de passe admin, envoie les images sur la plateforme, puis crée l'article, ou le met à jour s'il existe déjà. Il lit le Markdown avec le parseur du studio, dans le dépôt `lescracks-platform` voisin (sinon, indique son chemin avec `LESCRACKS_DIR`).
