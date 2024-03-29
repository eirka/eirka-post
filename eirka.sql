/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `acl_roles`
--

DROP TABLE IF EXISTS `acl_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `acl_roles` (
  `role_id` int unsigned NOT NULL AUTO_INCREMENT,
  `role_name` varchar(30) COLLATE utf8mb3_unicode_ci NOT NULL,
  `role_global` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`role_id`),
  UNIQUE KEY `role_name` (`role_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO acl_roles VALUES (1,"GUEST",1);
INSERT INTO acl_roles VALUES (2,"USER",1);
INSERT INTO acl_roles VALUES (3,"MODERATOR",0);
INSERT INTO acl_roles VALUES (4,"ADMIN",0);

--
-- Table structure for table `analytics`
--

DROP TABLE IF EXISTS `analytics`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `analytics` (
  `analytics_id` int NOT NULL AUTO_INCREMENT,
  `ib_id` tinyint unsigned NOT NULL,
  `user_id` int unsigned NOT NULL,
  `request_time` datetime NOT NULL,
  `request_ip` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `request_path` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `request_itemkey` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL DEFAULT '',
  `request_itemvalue` int unsigned NOT NULL DEFAULT '0',
  `request_cached` tinyint unsigned DEFAULT NULL,
  `request_status` tinyint unsigned NOT NULL,
  `request_latency` int unsigned NOT NULL,
  PRIMARY KEY (`analytics_id`),
  KEY `analytics_ib_id` (`ib_id`),
  KEY `analytics_user_id` (`user_id`),
  KEY `analytics_popular` (`ib_id`,`request_itemkey`,`request_time`),
  KEY `analytics_ib_time` (`ib_id`,`request_time`),
  CONSTRAINT `analytics_ib_id` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `analytics_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `audit`
--

DROP TABLE IF EXISTS `audit`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `audit` (
  `audit_id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `ib_id` tinyint unsigned NOT NULL,
  `audit_type` tinyint unsigned NOT NULL DEFAULT '1',
  `audit_ip` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `audit_time` datetime NOT NULL,
  `audit_action` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `audit_info` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  PRIMARY KEY (`audit_id`),
  KEY `audit_user_id_idx` (`user_id`),
  KEY `audit_ib_id_idx` (`ib_id`),
  CONSTRAINT `audit_ib_id` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `audit_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `banned_files`
--

DROP TABLE IF EXISTS `banned_files`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `banned_files` (
  `user_id` int unsigned NOT NULL,
  `ib_id` tinyint unsigned NOT NULL,
  `ban_hash` varchar(32) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ban_reason` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  UNIQUE KEY `bf_ban_hash` (`ban_hash`),
  KEY `bf_user_id` (`user_id`),
  KEY `bf_ib_id` (`ib_id`),
  CONSTRAINT `bf_ib_id` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `bf_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `banned_ips`
--

DROP TABLE IF EXISTS `banned_ips`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `banned_ips` (
  `user_id` int unsigned NOT NULL,
  `ib_id` tinyint unsigned NOT NULL,
  `ban_ip` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ban_reason` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  UNIQUE KEY `bip_ban_ip` (`ban_ip`),
  KEY `bip_user_id` (`user_id`),
  KEY `bip_ib_id` (`ib_id`),
  CONSTRAINT `bip_ib_id` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `bip_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `favorites`
--

DROP TABLE IF EXISTS `favorites`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `favorites` (
  `favorite_id` int unsigned NOT NULL AUTO_INCREMENT,
  `image_id` int unsigned NOT NULL,
  `user_id` int unsigned NOT NULL,
  PRIMARY KEY (`favorite_id`),
  UNIQUE KEY `favorites_uniq_user_image` (`user_id`,`image_id`),
  KEY `favorites_image_id` (`image_id`),
  KEY `favorites_user_id` (`user_id`),
  CONSTRAINT `favorites_image_id` FOREIGN KEY (`image_id`) REFERENCES `images` (`image_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `favorites_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `imageboards`
--

DROP TABLE IF EXISTS `imageboards`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `imageboards` (
  `ib_id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `ib_title` varchar(45) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_description` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_domain` varchar(40) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_nsfw` tinyint(1) NOT NULL DEFAULT '0',
  `ib_api` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_img` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_style` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_logo` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `ib_discord` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  PRIMARY KEY (`ib_id`),
  UNIQUE KEY `imageboards_uniq` (`ib_title`),
  KEY `ib_id_ib_title` (`ib_id`,`ib_title`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO imageboards VALUES (1,"Default","A default board","default.com",0,"default.com/api","default.com/img","default.css","logo.png","");

--
-- Table structure for table `images`
--

DROP TABLE IF EXISTS `images`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `images` (
  `image_id` int unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int unsigned NOT NULL,
  `image_file` varchar(20) COLLATE utf8mb3_unicode_ci NOT NULL,
  `image_thumbnail` varchar(20) COLLATE utf8mb3_unicode_ci NOT NULL,
  `image_hash` varchar(32) COLLATE utf8mb3_unicode_ci NOT NULL,
  `image_sha` char(40) COLLATE utf8mb3_unicode_ci NOT NULL,
  `image_orig_height` smallint unsigned NOT NULL DEFAULT '0',
  `image_orig_width` smallint unsigned NOT NULL DEFAULT '0',
  `image_tn_height` smallint unsigned NOT NULL DEFAULT '0',
  `image_tn_width` smallint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`image_id`),
  UNIQUE KEY `image_filename_uniq` (`image_file`),
  KEY `post_id_idx` (`post_id`),
  KEY `p_id_i_id` (`post_id`,`image_id`),
  KEY `hash_idx` (`image_hash`),
  KEY `image_sha_idx` (`image_sha`),
  CONSTRAINT `post_id` FOREIGN KEY (`post_id`) REFERENCES `posts` (`post_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `posts`
--

DROP TABLE IF EXISTS `posts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `posts` (
  `post_id` int unsigned NOT NULL AUTO_INCREMENT,
  `thread_id` smallint unsigned NOT NULL,
  `user_id` int unsigned NOT NULL DEFAULT '1',
  `post_deleted` tinyint(1) NOT NULL DEFAULT '0',
  `post_num` smallint unsigned NOT NULL DEFAULT '1',
  `post_ip` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `post_time` datetime NOT NULL,
  `post_text` text COLLATE utf8mb3_unicode_ci,
  PRIMARY KEY (`post_id`),
  KEY `thread_id_idx` (`thread_id`),
  KEY `t_id_p_id` (`thread_id`,`post_id`),
  KEY `posts_user_id` (`user_id`),
  CONSTRAINT `posts_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `thread_id` FOREIGN KEY (`thread_id`) REFERENCES `threads` (`thread_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `settings`
--

DROP TABLE IF EXISTS `settings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `settings` (
  `settings_key` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  `settings_value` varchar(255) COLLATE utf8mb3_unicode_ci NOT NULL,
  PRIMARY KEY (`settings_key`),
  UNIQUE KEY `settings_uniq` (`settings_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO settings VALUES ("akismet_host","");
INSERT INTO settings VALUES ("akismet_key","");
INSERT INTO settings VALUES ("amazon_bucket","");
INSERT INTO settings VALUES ("amazon_id","");
INSERT INTO settings VALUES ("amazon_key","");
INSERT INTO settings VALUES ("amazon_region","");
INSERT INTO settings VALUES ("auto_registration",1);
INSERT INTO settings VALUES ("guest_posting",1);
INSERT INTO settings VALUES ("comment_maxlength",1000);
INSERT INTO settings VALUES ("comment_minlength",3);
INSERT INTO settings VALUES ("image_maxheight",20000);
INSERT INTO settings VALUES ("image_maxsize",20000000);
INSERT INTO settings VALUES ("image_maxwidth",20000);
INSERT INTO settings VALUES ("image_minheight",100);
INSERT INTO settings VALUES ("image_minwidth",100);
INSERT INTO settings VALUES ("index_postsperthread",5);
INSERT INTO settings VALUES ("index_threadsperpage",10);
INSERT INTO settings VALUES ("name_maxlength",20);
INSERT INTO settings VALUES ("name_minlength",3);
INSERT INTO settings VALUES ("param_maxsize",1000000);
INSERT INTO settings VALUES ("prim_css","prim.css?1");
INSERT INTO settings VALUES ("prim_js","prim.js?1");
INSERT INTO settings VALUES ("sfs_confidence",40);
INSERT INTO settings VALUES ("tag_maxlength",128);
INSERT INTO settings VALUES ("tag_minlength",3);
INSERT INTO settings VALUES ("thread_postsmax",800);
INSERT INTO settings VALUES ("thread_postsperpage",50);
INSERT INTO settings VALUES ("thumbnail_maxheight",300);
INSERT INTO settings VALUES ("thumbnail_maxwidth",200);
INSERT INTO settings VALUES ("title_maxlength",40);
INSERT INTO settings VALUES ("title_minlength",3);
INSERT INTO settings VALUES ("webm_maxlength",300);
INSERT INTO settings VALUES ("avatar_minwidth",100);
INSERT INTO settings VALUES ("avatar_minheight",100);
INSERT INTO settings VALUES ("avatar_maxwidth",1000);
INSERT INTO settings VALUES ("avatar_maxheight",1000);
INSERT INTO settings VALUES ("avatar_maxsize",1000000);
INSERT INTO settings VALUES ("password_minlength",8);
INSERT INTO settings VALUES ("password_maxlength",128);
INSERT INTO settings VALUES ("cloudflare_key","");
INSERT INTO settings VALUES ("cloudflare_email","");
INSERT INTO settings VALUES ("scamalytics_endpoint","");
INSERT INTO settings VALUES ("scamalytics_key","");
INSERT INTO settings VALUES ("scamalytics_path","");
INSERT INTO settings VALUES ("scamalytics_score","60");

--
-- Table structure for table `tagmap`
--

DROP TABLE IF EXISTS `tagmap`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tagmap` (
  `image_id` int unsigned NOT NULL,
  `tag_id` int unsigned NOT NULL,
  PRIMARY KEY (`image_id`,`tag_id`),
  UNIQUE KEY `tagmap_uniq` (`image_id`,`tag_id`),
  KEY `image_id_idx` (`image_id`),
  KEY `tag_id_idx` (`tag_id`),
  KEY `tagmap_tag_i` (`image_id`,`tag_id`),
  CONSTRAINT `image_id` FOREIGN KEY (`image_id`) REFERENCES `images` (`image_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `tag_id` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`tag_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `tags`
--

DROP TABLE IF EXISTS `tags`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tags` (
  `tag_id` int unsigned NOT NULL AUTO_INCREMENT,
  `ib_id` tinyint unsigned NOT NULL,
  `tagtype_id` int unsigned NOT NULL,
  `tag_name` varchar(128) COLLATE utf8mb3_unicode_ci NOT NULL,
  PRIMARY KEY (`tag_id`),
  UNIQUE KEY `tags_uniq` (`ib_id`,`tag_name`),
  KEY `tags_ib_id_idx` (`ib_id`),
  KEY `tag_id_tag_name` (`tag_id`,`tag_name`),
  KEY `tagtype_id_idx` (`tagtype_id`),
  KEY `tt_t_id` (`tagtype_id`,`tag_id`),
  FULLTEXT KEY `tags_tag_name_idx` (`tag_name`),
  CONSTRAINT `ib_id_tags` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `tagtype_id` FOREIGN KEY (`tagtype_id`) REFERENCES `tagtype` (`tagtype_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `tagtype`
--

DROP TABLE IF EXISTS `tagtype`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tagtype` (
  `tagtype_id` int unsigned NOT NULL AUTO_INCREMENT,
  `tagtype_name` varchar(45) COLLATE utf8mb3_unicode_ci NOT NULL,
  PRIMARY KEY (`tagtype_id`),
  KEY `tt_id_tt_name` (`tagtype_id`,`tagtype_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO tagtype VALUES (1,"Tag");
INSERT INTO tagtype VALUES (2,"Artist");
INSERT INTO tagtype VALUES (3,"Character");
INSERT INTO tagtype VALUES (4,"Copyright");

--
-- Table structure for table `threads`
--

DROP TABLE IF EXISTS `threads`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `threads` (
  `thread_id` smallint unsigned NOT NULL AUTO_INCREMENT,
  `ib_id` tinyint unsigned NOT NULL,
  `thread_title` varchar(45) COLLATE utf8mb3_unicode_ci NOT NULL,
  `thread_closed` tinyint(1) NOT NULL DEFAULT '0',
  `thread_sticky` tinyint(1) NOT NULL DEFAULT '0',
  `thread_deleted` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`thread_id`),
  KEY `ib_id_idx` (`ib_id`),
  KEY `t_id_ib_id` (`ib_id`,`thread_id`),
  FULLTEXT KEY `threads_thread_title_idx` (`thread_title`),
  CONSTRAINT `ib_id` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_ib_role_map`
--

DROP TABLE IF EXISTS `user_ib_role_map`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_ib_role_map` (
  `user_id` int unsigned NOT NULL,
  `ib_id` tinyint unsigned NOT NULL,
  `role_id` int unsigned NOT NULL,
  UNIQUE KEY `uirm_uniq` (`user_id`,`ib_id`,`role_id`),
  KEY `uirm_user_id` (`user_id`),
  KEY `uirm_ib_id` (`ib_id`),
  KEY `uirm_role_id` (`role_id`),
  CONSTRAINT `uirm_ib_id` FOREIGN KEY (`ib_id`) REFERENCES `imageboards` (`ib_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `uirm_role_id` FOREIGN KEY (`role_id`) REFERENCES `acl_roles` (`role_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `uirm_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_role_map`
--

DROP TABLE IF EXISTS `user_role_map`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_role_map` (
  `user_id` int unsigned NOT NULL,
  `role_id` int unsigned NOT NULL,
  UNIQUE KEY `urm_uniq` (`user_id`,`role_id`),
  KEY `urm_role_id` (`role_id`),
  KEY `urm_user_id` (`user_id`),
  CONSTRAINT `urm_role_id` FOREIGN KEY (`role_id`) REFERENCES `acl_roles` (`role_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `urm_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

INSERT INTO user_role_map VALUES (1,1);

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `user_id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_name` varchar(20) COLLATE utf8mb3_unicode_ci NOT NULL,
  `user_email` varchar(255) COLLATE utf8mb3_unicode_ci DEFAULT NULL,
  `user_password` binary(60) DEFAULT NULL,
  `user_confirmed` tinyint(1) NOT NULL DEFAULT '0',
  `user_banned` tinyint(1) NOT NULL DEFAULT '0',
  `user_locked` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `user_name` (`user_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_unicode_ci;

INSERT INTO users VALUES (1,"Anonymous",NULL,NULL,0,0,0);

/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
